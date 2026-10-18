package access

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"scabbage/internal/store"
)

type fixture struct {
	svc     *Service
	now     time.Time
	changes int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	f := &fixture{now: time.Unix(1_700_000_000, 0)}
	f.svc = New(db, Options{
		Now:             func() time.Time { return f.now },
		OnPendingChange: func() { f.changes++ },
	})
	return f
}

func secret(b byte) []byte { return bytes.Repeat([]byte{b}, SecretBytes) }

func (f *fixture) code(t *testing.T, uses int) string {
	t.Helper()
	c, err := f.svc.CreateCode(uses, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return c.Code
}

func (f *fixture) pending(t *testing.T, s []byte, name string) Result {
	t.Helper()
	r, err := f.svc.RequestJoin(s, Join{Code: f.code(t, 1), Name: name, Color: 0xff0000}, "1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != StatusPending {
		t.Fatalf("status = %v, want pending", r.Status)
	}
	return r
}

func TestUnknownDeviceIsRejected(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.Check(secret(1)); !errors.Is(err, ErrUnknownDevice) {
		t.Fatalf("err = %v, want ErrUnknownDevice", err)
	}
}

func TestShortSecretIsRejected(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.Check([]byte("short")); !errors.Is(err, ErrBadSecret) {
		t.Fatalf("err = %v, want ErrBadSecret", err)
	}
}

func TestJoinThenApproveGrantsDevice(t *testing.T) {
	f := newFixture(t)
	r := f.pending(t, secret(1), "Ann")
	if got, _ := f.svc.Check(secret(1)); got.Status != StatusPending || got.RequestID != r.RequestID {
		t.Fatalf("check while pending = %+v", got)
	}

	ch, cancel := f.svc.Await(r.RequestID)
	defer cancel()
	if err := f.svc.Decide(r.RequestID, true); err != nil {
		t.Fatal(err)
	}
	if approved := <-ch; !approved {
		t.Fatal("waiter got deny, want approve")
	}

	got, err := f.svc.Check(secret(1))
	if err != nil || got.Status != StatusApproved {
		t.Fatalf("check after approve = %+v, %v", got, err)
	}
	if got.Player.Name != "Ann" || got.Player.Color != 0xff0000 || got.Player.Admin {
		t.Fatalf("player = %+v", got.Player)
	}
}

func TestDenyBlocksDevice(t *testing.T) {
	f := newFixture(t)
	r := f.pending(t, secret(1), "Ann")
	if err := f.svc.Decide(r.RequestID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Check(secret(1)); !errors.Is(err, ErrDenied) {
		t.Fatalf("err = %v, want ErrDenied", err)
	}
	if err := f.svc.Decide(r.RequestID, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second decision err = %v, want ErrNotFound", err)
	}
}

func TestRepeatJoinReusesPendingRequest(t *testing.T) {
	f := newFixture(t)
	code := f.code(t, 1)
	j := Join{Code: code, Name: "Ann", Color: 1}
	a, err := f.svc.RequestJoin(secret(1), j, "ip")
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.svc.RequestJoin(secret(1), j, "ip")
	if err != nil || b.RequestID != a.RequestID {
		t.Fatalf("repeat = %+v, %v; want same request without using the code again", b, err)
	}
}

func TestCodeUsesAndExpiry(t *testing.T) {
	f := newFixture(t)
	code := f.code(t, 1)
	if _, err := f.svc.RequestJoin(secret(1), Join{Code: code, Name: "Ann"}, "ip"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.RequestJoin(secret(2), Join{Code: code, Name: "Bob"}, "ip"); !errors.Is(err, ErrBadCode) {
		t.Fatalf("used-up code err = %v, want ErrBadCode", err)
	}

	expiring := f.code(t, 5)
	f.now = f.now.Add(2 * time.Hour)
	if _, err := f.svc.RequestJoin(secret(3), Join{Code: expiring, Name: "Cy"}, "ip"); !errors.Is(err, ErrBadCode) {
		t.Fatalf("expired code err = %v, want ErrBadCode", err)
	}
}

func TestCodeIsNormalized(t *testing.T) {
	f := newFixture(t)
	code := f.code(t, 1)
	messy := " " + code[:4] + "-" + string(bytes.ToLower([]byte(code[4:]))) + " "
	if _, err := f.svc.RequestJoin(secret(1), Join{Code: messy, Name: "Ann"}, "ip"); err != nil {
		t.Fatalf("messy code rejected: %v", err)
	}
}

func TestBadCodesAreRateLimitedPerIP(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < maxCodeFailures; i++ {
		if _, err := f.svc.RequestJoin(secret(1), Join{Code: "NOPENOPE", Name: "Ann"}, "6.6.6.6"); !errors.Is(err, ErrBadCode) {
			t.Fatalf("attempt %d err = %v, want ErrBadCode", i, err)
		}
	}
	good := f.code(t, 1)
	if _, err := f.svc.RequestJoin(secret(1), Join{Code: good, Name: "Ann"}, "6.6.6.6"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited even with a good code", err)
	}
	if _, err := f.svc.RequestJoin(secret(2), Join{Code: good, Name: "Bob"}, "7.7.7.7"); err != nil {
		t.Fatalf("other IP should not be limited: %v", err)
	}
	f.now = f.now.Add(failureWindow)
	if _, err := f.svc.RequestJoin(secret(1), Join{Code: "NOPENOPE", Name: "Ann"}, "6.6.6.6"); !errors.Is(err, ErrBadCode) {
		t.Fatalf("after window err = %v, want ErrBadCode", err)
	}
}

func TestPendingRequestsAreCapped(t *testing.T) {
	f := newFixture(t)
	code := f.code(t, maxPending+1)
	for i := 0; i < maxPending; i++ {
		if _, err := f.svc.RequestJoin(secret(byte(i+1)), Join{Code: code, Name: "P"}, "ip"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.svc.RequestJoin(secret(200), Join{Code: code, Name: "P"}, "ip"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}
}

func TestPendingRequestsExpire(t *testing.T) {
	f := newFixture(t)
	f.pending(t, secret(1), "Ann")
	f.now = f.now.Add(pendingTTL + time.Second)
	if list, _ := f.svc.Pending(); len(list) != 0 {
		t.Fatalf("pending = %v, want expired", list)
	}
	if _, err := f.svc.Check(secret(1)); !errors.Is(err, ErrUnknownDevice) {
		t.Fatalf("err = %v, want ErrUnknownDevice for expired request", err)
	}
	f.pending(t, secret(1), "Ann") // the same device may ask again
}

func TestInvalidNamesAreRejected(t *testing.T) {
	f := newFixture(t)
	for _, name := range []string{"", "   ", "this name is far too long", "<script>", "a\x00b"} {
		if _, err := f.svc.RequestJoin(secret(1), Join{Code: f.code(t, 1), Name: name}, "ip"); !errors.Is(err, ErrBadName) {
			t.Errorf("name %q err = %v, want ErrBadName", name, err)
		}
	}
}

func TestReturningPlayerLinksNewDevice(t *testing.T) {
	f := newFixture(t)
	first := f.pending(t, secret(1), "Ann")
	if err := f.svc.Decide(first.RequestID, true); err != nil {
		t.Fatal(err)
	}
	second := f.pending(t, secret(2), "ann")

	list, err := f.svc.Pending()
	if err != nil || len(list) != 1 || !list[0].Returning {
		t.Fatalf("pending = %+v, %v; want one returning request", list, err)
	}
	if err := f.svc.Decide(second.RequestID, true); err != nil {
		t.Fatal(err)
	}
	a, _ := f.svc.Check(secret(1))
	b, _ := f.svc.Check(secret(2))
	if a.Player.ID != b.Player.ID {
		t.Fatalf("devices map to players %d and %d, want the same", a.Player.ID, b.Player.ID)
	}
}

func TestAdminInviteAutoApprovesAdmin(t *testing.T) {
	f := newFixture(t)
	code, ok, err := f.svc.AdminInvite()
	if err != nil || !ok {
		t.Fatalf("AdminInvite = %q, %v, %v", code, ok, err)
	}
	again, _, _ := f.svc.AdminInvite()
	if again != code {
		t.Fatalf("second invite %q, want reuse of %q", again, code)
	}

	r, err := f.svc.RequestJoin(secret(1), Join{Code: code, Name: "Mom", Color: 2}, "ip")
	if err != nil || r.Status != StatusApproved || !r.Player.Admin {
		t.Fatalf("admin join = %+v, %v", r, err)
	}
	if _, ok, _ := f.svc.AdminInvite(); ok {
		t.Fatal("no admin invite should be offered once an admin exists")
	}
}

func TestPendingChangesNotify(t *testing.T) {
	f := newFixture(t)
	r := f.pending(t, secret(1), "Ann")
	if f.changes != 1 {
		t.Fatalf("changes after request = %d, want 1", f.changes)
	}
	_ = f.svc.Decide(r.RequestID, true)
	if f.changes != 2 {
		t.Fatalf("changes after decision = %d, want 2", f.changes)
	}
}

func TestAwaitCancelStopsDelivery(t *testing.T) {
	f := newFixture(t)
	r := f.pending(t, secret(1), "Ann")
	ch, cancel := f.svc.Await(r.RequestID)
	cancel()
	_ = f.svc.Decide(r.RequestID, true)
	select {
	case <-ch:
		t.Fatal("cancelled waiter should not receive")
	default:
	}
}

func TestCodesUseUnambiguousAlphabet(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 50; i++ {
		c := f.code(t, 1)
		if len(c) != codeLength {
			t.Fatalf("code %q length %d", c, len(c))
		}
		if bytes.ContainsAny([]byte(c), "01IOL") {
			t.Fatalf("code %q has ambiguous characters", c)
		}
	}
}
