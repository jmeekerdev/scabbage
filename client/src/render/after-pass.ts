import {
  BufferGeometry,
  Color,
  CustomBlending,
  DepthTexture,
  Float32BufferAttribute,
  Fog,
  Mesh,
  NormalBlending,
  OrthographicCamera,
  PerspectiveCamera,
  Scene,
  ShaderMaterial,
  SrcColorFactor,
  SRGBColorSpace,
  Vector2,
  WebGLRenderTarget,
  ZeroFactor,
  type WebGLRenderer,
} from "three";
import type { Look } from "../looks/look";

/** The afterwards features a look switches on, as defines, or undefined when it needs no pass at all. */
export function afterFeatures(look: Look, overhead: boolean): Record<string, string> | undefined {
  const a = look.after ?? {};
  const on = {
    OUTLINE: !!a.outline && a.outline.strength > 0,
    CREASES: !!a.outline?.creases,
    DARKEN: !!a.outline && !a.outline.color,
    SOFTEN: (a.soften ?? 0) > 0,
    PAPER: (a.paper ?? 0) > 0,
    TILT: overhead && (a.tiltShift ?? 0) > 0,
    VIGNETTE: (a.vignette ?? 0) > 0,
    SATURATE: (a.saturation ?? 1) !== 1,
  };
  if (!on.OUTLINE) on.CREASES = on.DARKEN = false;
  const defines = Object.entries(on).flatMap(([k, v]) => (v ? [[k, ""]] : []));
  if (!defines.length) return undefined;
  // A vignette alone needs no copy of the frame: it is laid over it, darkening what is there.
  if (defines.length === 1 && on.VIGNETTE) defines.push(["OVERLAY", ""]);
  return Object.fromEntries(defines);
}

const vertexShader = /* glsl */ `
varying vec2 vUv;
void main() {
  vUv = position.xy * 0.5 + 0.5;
  gl_Position = vec4(position.xy, 0.0, 1.0);
}`;

const fragmentShader = /* glsl */ `
uniform sampler2D tColor;
uniform sampler2D tDepth;
uniform vec2 res;
uniform float near;
uniform float far;
uniform float fogNear;
uniform float fogFar;
uniform vec3 lineColor;
uniform float lineStrength;
uniform float lineWidth;
uniform float soften;
uniform float paper;
uniform float tilt;
uniform float vignette;
uniform float saturation;
varying vec2 vUv;

float hash(vec2 p) {
  vec3 q = fract(vec3(p.xyx) * 0.1031);
  q += dot(q, q.yzx + 33.33);
  return fract((q.x + q.y) * q.z);
}
float noise(vec2 p) {
  vec2 i = floor(p), f = fract(p);
  f = f * f * (3.0 - 2.0 * f);
  return mix(mix(hash(i), hash(i + vec2(1, 0)), f.x), mix(hash(i + vec2(0, 1)), hash(i + vec2(1, 1)), f.x), f.y);
}
// Distance along the view, and its inverse, which changes evenly across any flat surface on screen.
float viewZ(vec2 uv) { return near * far / (far - texture2D(tDepth, uv).r * (far - near)); }

// How much depth bends here: 0 across flat surfaces, large where one thing stands in front of another.
float bend(vec2 uv, vec2 step, float w0) {
  return abs(1.0 / viewZ(uv + step) + 1.0 / viewZ(uv - step) - 2.0 * w0) / w0;
}

float line(vec2 uv) {
  vec2 px = vec2(lineWidth * res.y / 750.0) / res;
  float z0 = viewZ(uv), w0 = 1.0 / z0;
  float b = max(bend(uv, vec2(px.x, 0.0), w0), bend(uv, vec2(0.0, px.y), w0));
  #ifdef CREASES
  float e = smoothstep(0.004, 0.012, b);
  #else
  float e = smoothstep(0.04, 0.1, b);
  #endif
  return e * (1.0 - smoothstep(fogNear, fogFar, z0));
}

vec3 blurred(vec2 uv, float r) {
  vec3 sum = vec3(0.0);
  for (int i = 0; i < 16; i++) {
    float t = float(i) * 2.39996, d = sqrt((float(i) + 0.5) / 16.0) * r;
    sum += texture2D(tColor, uv + vec2(cos(t), sin(t)) * d / res).rgb;
  }
  return sum / 16.0;
}

float vignetted() {
  vec2 q = vUv - 0.5;
  return 1.0 - vignette * dot(q, q) * 1.6;
}

void main() {
  #ifdef OVERLAY
  gl_FragColor = vec4(vec3(vignetted()), 1.0);
  #include <colorspace_fragment>
  return;
  #endif
  vec3 col = texture2D(tColor, vUv).rgb;
  #ifdef TILT
  float r = tilt * smoothstep(0.1, 0.5, abs(vUv.y - 0.52)) * res.y / 110.0;
  if (r > 0.3) col = blurred(vUv, r);
  #endif
  #ifdef SOFTEN
  col = mix(col, blurred(vUv, soften * res.y / 700.0), 0.5);
  #endif
  #ifdef OUTLINE
  #ifdef DARKEN
  vec3 ink = col * 0.5;
  #else
  vec3 ink = lineColor;
  #endif
  col = mix(col, ink, line(vUv) * lineStrength);
  #endif
  #ifdef SATURATE
  col = mix(vec3(dot(col, vec3(0.2126, 0.7152, 0.0722))), col, saturation);
  #endif
  #ifdef PAPER
  float grain = hash(floor(vUv * res)), blot = noise(vUv * vec2(7.0, 5.0));
  float fiber = noise(vUv * res * vec2(0.04, 0.5));
  col *= 1.0 - paper * (0.05 * grain + 0.08 * blot + 0.04 * fiber);
  #endif
  #ifdef VIGNETTE
  col *= vignetted();
  #endif
  gl_FragColor = vec4(col, 1.0);
  #include <colorspace_fragment>
}`;

/** One triangle that covers the whole screen. */
function screenTriangle(): BufferGeometry {
  const g = new BufferGeometry();
  g.setAttribute("position", new Float32BufferAttribute([-1, -1, 0, 3, -1, 0, -1, 3, 0], 3));
  return g;
}

/**
 * Draws the scene into a picture with depth, then onto the screen through the look's afterwards
 * effects. Holds no picture while the look has none, so plain looks cost nothing, and a lone
 * vignette is laid over the frame instead.
 */
export class AfterPass {
  private target?: WebGLRenderTarget;
  private on = false;
  private overlay = false;
  private readonly size = new Vector2();
  private readonly flat = new OrthographicCamera();
  private readonly screen = new Scene();
  private readonly material = new ShaderMaterial({
    vertexShader,
    fragmentShader,
    depthTest: false,
    depthWrite: false,
    blendSrc: ZeroFactor, // used while overlaid: the frame times the shader's output
    blendDst: SrcColorFactor,
    uniforms: Object.fromEntries(
      [
        ["tColor", null],
        ["tDepth", null],
        ["res", new Vector2()],
        ["near", 0],
        ["far", 1],
        ["fogNear", 0],
        ["fogFar", 1],
        ["lineColor", new Color()],
        ["lineStrength", 0],
        ["lineWidth", 1],
        ["soften", 0],
        ["paper", 0],
        ["tilt", 0],
        ["vignette", 0],
        ["saturation", 1],
      ].map(([k, value]) => [k, { value }]),
    ),
  });

  /** samples: antialiasing samples for the scene picture, 0 for none. */
  constructor(private readonly samples: number) {
    const tri = new Mesh(screenTriangle(), this.material);
    tri.frustumCulled = false;
    this.screen.add(tri);
  }

  /** Whether a pass runs at all. */
  get active(): boolean {
    return this.on;
  }

  /** Takes the look's features and numbers; dropping the pass also frees its picture. */
  setLook(look: Look, overhead: boolean): void {
    const defines = afterFeatures(look, overhead);
    const m = this.material;
    this.on = !!defines;
    this.overlay = !!defines && "OVERLAY" in defines;
    m.blending = this.overlay ? CustomBlending : NormalBlending;
    if (!defines || this.overlay) {
      this.target?.dispose();
      this.target = undefined;
    }
    if (!defines) return;
    if (JSON.stringify(defines) !== JSON.stringify(m.defines)) {
      m.defines = defines;
      m.needsUpdate = true;
    }
    const a = look.after ?? {};
    const u = m.uniforms;
    const c = a.outline?.color;
    if (c) (u.lineColor.value as Color).setRGB(c.r, c.g, c.b, SRGBColorSpace);
    u.lineStrength.value = a.outline?.strength ?? 0;
    u.lineWidth.value = a.outline?.width ?? 1;
    u.soften.value = a.soften ?? 0;
    u.paper.value = a.paper ?? 0;
    u.tilt.value = a.tiltShift ?? 0;
    u.vignette.value = a.vignette ?? 0;
    u.saturation.value = a.saturation ?? 1;
  }

  /** Draws scene through the pass onto the screen. */
  render(renderer: WebGLRenderer, scene: Scene, camera: PerspectiveCamera): void {
    if (this.overlay) return this.overlaid(renderer, scene, camera);
    const target = this.fit(renderer);
    const u = this.material.uniforms;
    const fog = scene.fog as Fog | null;
    u.tColor.value = target.texture;
    u.tDepth.value = target.depthTexture;
    (u.res.value as Vector2).copy(this.size);
    u.near.value = camera.near;
    u.far.value = camera.far;
    u.fogNear.value = fog?.near ?? camera.far;
    u.fogFar.value = fog?.far ?? camera.far;
    renderer.setRenderTarget(target);
    renderer.render(scene, camera);
    renderer.setRenderTarget(null);
    renderer.render(this.screen, this.flat);
  }

  /** Draws scene straight to the screen, then multiplies the pass's output over it. */
  private overlaid(renderer: WebGLRenderer, scene: Scene, camera: PerspectiveCamera): void {
    renderer.render(scene, camera);
    const autoClear = renderer.autoClear;
    renderer.autoClear = false;
    renderer.render(this.screen, this.flat);
    renderer.autoClear = autoClear;
  }

  /** The scene picture, remade whenever the drawing size changes. */
  private fit(renderer: WebGLRenderer): WebGLRenderTarget {
    const { x, y } = renderer.getDrawingBufferSize(this.size);
    if (this.target?.width === x && this.target.height === y) return this.target;
    this.target?.dispose();
    this.target = new WebGLRenderTarget(x, y, {
      samples: this.samples,
      depthTexture: new DepthTexture(x, y),
    });
    this.target.texture.colorSpace = SRGBColorSpace;
    return this.target;
  }
}
