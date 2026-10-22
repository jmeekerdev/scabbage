import { describe, expect, it } from "vitest";
import { glslErrors } from "./glsl-check";

const vertexShader = "void main() { gl_Position = projectionMatrix * vec4(position, 1.0); }";

describe("glslErrors", () => {
  it("passes a shader three would compile", () => {
    const fragmentShader = "void main() { gl_FragColor = vec4(1.0); }";
    expect(glslErrors({ vertexShader, fragmentShader })).toEqual([]);
  });

  it("resolves three's includes and the material's defines", () => {
    const fragmentShader = `#include <common>
      void main() { gl_FragColor = vec4(vec3(SCALE * PI), 1.0); }`;
    expect(glslErrors({ vertexShader, fragmentShader, defines: { SCALE: "2.0" } })).toEqual([]);
  });

  it("names the stage and line of a mistake", () => {
    const fragmentShader = "void main() { gl_FragColor = vec4(1.0) * colour; }";
    const errors = glslErrors({ vertexShader, fragmentShader });
    expect(errors).toHaveLength(1);
    expect(errors[0]).toMatch(/^fragment: .*'colour'.*undeclared/);
  });
});
