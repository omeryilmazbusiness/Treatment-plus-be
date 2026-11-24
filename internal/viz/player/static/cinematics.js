import * as THREE from "three";

const EASE = "cubic-bezier(0.22, 1, 0.36, 1)";
const OPEN_ANGLE = 0.12;
const HINGE_NAME = "jaw_hinge_pivot";

function pick(root, ...names) {
  for (const name of names) {
    if (!name) continue;
    const hit = root.getObjectByName(name);
    if (hit) return hit;
    let found = null;
    const key = String(name).toLowerCase();
    root.traverse((o) => {
      if (!found && o.name && o.name.toLowerCase().includes(key)) found = o;
    });
    if (found) return found;
  }
  console.warn("[viz] mesh not found:", names.filter(Boolean).join(", "));
  return null;
}

function boxOf(obj) {
  const box = new THREE.Box3().setFromObject(obj);
  return {
    box,
    size: box.getSize(new THREE.Vector3()),
    center: box.getCenter(new THREE.Vector3()),
    radius: box.getSize(new THREE.Vector3()).length() * 0.5,
  };
}

function fdiMeta(fdi) {
  const n = Number(fdi) || 16;
  const q = Math.floor(n / 10);
  return {
    n,
    upper: q === 1 || q === 2 || q === 5 || q === 6,
    right: q === 1 || q === 4 || q === 5 || q === 8,
    distal: Math.max(0, Math.min(1, ((n % 10) - 1) / 7)),
  };
}

export function archTypeOf(fdi) {
  return fdiMeta(fdi).upper ? "upper" : "lower";
}

function archMesh(root, fdi) {
  const upper = archTypeOf(fdi) === "upper";
  return upper
    ? pick(root, "TeethUpper_midres", "TeethUpper", "Tooth_Upper_Molar_16")
    : pick(root, "TeethLower_midres", "TeethLower", "Tooth_Lower_Molar_36");
}

function gumsMesh(root, fdi) {
  const upper = archTypeOf(fdi) === "upper";
  return upper
    ? pick(root, "GumsUpper_midres", "GumsUpper")
    : pick(root, "GumsLower_midres", "GumsLower");
}

export function archApproachLabel(fdi) {
  const { n, upper } = fdiMeta(fdi);
  return upper
    ? `Diş ${n} maksilla · ağız tabanından yukarı, damak örtmez`
    : `Diş ${n} mandibula · üstten aşağı, dil örtmez`;
}

function fdiPoint(mesh, fdi) {
  const { box, center, size } = boxOf(mesh);
  const { upper, right, distal } = fdiMeta(fdi);
  const p = new THREE.Vector3();
  p.x = center.x + (right ? -1 : 1) * (0.18 + 0.38 * distal) * size.x;
  p.y = upper ? box.min.y + size.y * 0.12 : box.max.y - size.y * 0.12;
  p.z = box.max.z - Math.pow(distal, 0.82) * size.z * 0.96;
  return p;
}

function eachMat(obj, fn) {
  if (!obj) return;
  obj.traverse((n) => {
    if (!n.isMesh) return;
    const list = Array.isArray(n.material) ? n.material : [n.material];
    list.forEach((m) => m && fn(m, n));
  });
}

function lookCam(ctx, pos, target, dur, tl, at, { keepLocked = true } = {}) {
  const t0 = at ?? tl.duration();
  const proxy = { u: 0 };
  const fromPos = new THREE.Vector3();
  const fromLook = new THREE.Vector3();
  const fromDir = new THREE.Vector3();
  const toPos = pos.clone();
  const toLook = target.clone();
  const tmpLook = new THREE.Vector3();
  const tmpOff = new THREE.Vector3();
  const toDir = new THREE.Vector3();
  const q0 = new THREE.Quaternion();
  const q1 = new THREE.Quaternion();
  const midDir = new THREE.Vector3(0, 0, 1);
  const state = { fromR: 1 };

  tl.add(() => { ctx.controls.enabled = false; }, t0);
  tl.fromTo(proxy, { u: 0 }, {
    u: 1,
    duration: dur,
    ease: EASE,
    immediateRender: false,
    onStart: () => {
      fromPos.copy(ctx.camera.position);
      fromLook.copy(ctx.controls.target);
      tmpOff.copy(fromPos).sub(fromLook);
      state.fromR = Math.max(tmpOff.length(), 1e-5);
      fromDir.copy(tmpOff).multiplyScalar(1 / state.fromR);
      ctx.controls.enabled = false;
    },
    onUpdate: () => {
      const u = proxy.u;
      tmpLook.copy(fromLook).lerp(toLook, u);
      tmpOff.copy(toPos).sub(toLook);
      const toR = Math.max(tmpOff.length(), 1e-5);
      toDir.copy(tmpOff).multiplyScalar(1 / toR);
      const dot = fromDir.dot(toDir);
      if (dot > 0.9995) {
        tmpOff.copy(fromDir);
      } else if (dot < -0.999) {
        const a = u < 0.5 ? fromDir : midDir;
        const b = u < 0.5 ? midDir : toDir;
        const s = u < 0.5 ? u * 2 : (u - 0.5) * 2;
        q1.setFromUnitVectors(a, b);
        q0.identity().slerp(q1, s);
        tmpOff.copy(a).applyQuaternion(q0);
      } else {
        q1.setFromUnitVectors(fromDir, toDir);
        q0.identity().slerp(q1, u);
        tmpOff.copy(fromDir).applyQuaternion(q0);
      }
      const r = state.fromR + (toR - state.fromR) * u;
      ctx.camera.position.copy(tmpLook).addScaledVector(tmpOff, r);
      ctx.controls.target.copy(tmpLook);
      ctx.camera.lookAt(tmpLook);
      if (ctx.bokeh?.uniforms?.focus) ctx.bokeh.uniforms.focus.value = r;
    },
  }, t0);
  tl.add(() => {
    ctx.camera.position.copy(toPos);
    ctx.controls.target.copy(toLook);
    ctx.camera.lookAt(toLook);
    if (ctx.bokeh?.uniforms?.focus) ctx.bokeh.uniforms.focus.value = toPos.distanceTo(toLook);
    if (!keepLocked) ctx.controls.enabled = true;
  }, t0 + dur);
}

function setDoF(ctx, focus, blur, { aperture, duration = 1.05, tl, at } = {}) {
  if (!ctx.bokeh?.uniforms) return;
  const u = ctx.bokeh.uniforms;
  const apply = (obj, value) => {
    if (!obj || value == null) return;
    const vars = { value, duration, ease: EASE };
    if (tl) tl.to(obj, vars, at ?? tl.duration());
    else ctx.gsap.to(obj, vars);
  };
  apply(u.focus, focus);
  apply(u.maxblur, blur);
  apply(u.aperture, aperture);
}

function createHinge(jaw) {
  if (jaw.userData.hinge) return jaw.userData.hinge;
  if (jaw.parent?.name === HINGE_NAME) {
    jaw.userData.hinge = jaw.parent;
    return jaw.parent;
  }
  const parent = jaw.parent;
  if (!parent) {
    console.warn("[viz] jawLower has no parent — cannot build TMJ hinge");
    return jaw;
  }
  const lower = pick(jaw, "GumsLower_midres", "TeethLower_midres") || jaw;
  const { box, size } = boxOf(lower);
  const hingeWorld = new THREE.Vector3(
    (box.min.x + box.max.x) * 0.5,
    box.min.y + size.y * 0.42,
    box.min.z - size.z * 0.55,
  );
  const pivot = new THREE.Group();
  pivot.name = HINGE_NAME;
  parent.add(pivot);
  parent.worldToLocal(hingeWorld);
  pivot.position.copy(hingeWorld);
  pivot.attach(jaw);
  jaw.userData.hinge = pivot;
  return pivot;
}

export function ensureOpenMouth(root, gsap, angle = OPEN_ANGLE, duration = 1.15) {
  const jaw = pick(root, "jawLower_midres", "jawLower");
  if (!jaw) {
    console.warn("[viz] jawLower_midres not found — mouth stays as authored");
    return gsap.timeline();
  }
  const pivot = createHinge(jaw);
  const tl = gsap.timeline({ defaults: { ease: EASE } });
  if (duration <= 0) {
    pivot.rotation.x = angle;
    return tl;
  }
  return tl.to(pivot.rotation, { x: angle, duration });
}

function toothWorldPosition(toothMeshOrPos, fdi) {
  const p = new THREE.Vector3();
  if (toothMeshOrPos?.isVector3) return p.copy(toothMeshOrPos);
  if (toothMeshOrPos && fdi != null) return fdiPoint(toothMeshOrPos, fdi);
  if (toothMeshOrPos?.getWorldPosition) return toothMeshOrPos.getWorldPosition(p);
  if (toothMeshOrPos?.x != null) return p.copy(toothMeshOrPos);
  return p;
}

function actionDistance(actionType) {
  if (actionType === "overview") return 0.22;
  if (actionType === "section") return 0.08;
  return 0.068;
}

export function getOptimalCameraPosition(toothMesh, archType, opts = {}) {
  const fdi = opts.fdi ?? 16;
  const upper = (archType || archTypeOf(fdi)) === "upper";
  const actionType = opts.actionType || "occlusal";
  const toothPosition = toothWorldPosition(toothMesh, fdi);
  const offset = new THREE.Vector3();
  if (upper) offset.set(0, -15, 25);
  else offset.set(0, 15, 25);
  if (actionType === "section") {
    const buccal = fdiMeta(fdi).right ? -1 : 1;
    offset.x = buccal * 5;
    offset.y *= 0.88;
  }
  const dir = offset.normalize();
  const dist = actionDistance(actionType);
  return {
    targetCameraPosition: toothPosition.clone().addScaledVector(dir, dist),
    targetLookAt: toothPosition.clone(),
    dist,
    dir,
    archType: upper ? "upper" : "lower",
    toothPosition,
  };
}

export function positionCameraForTooth(ctx, toothNumber, actionType = "occlusal", opts = {}) {
  const fdi = Number(toothNumber) || 16;
  const mesh = archMesh(ctx.root, fdi) || ctx.root;
  const framed = getOptimalCameraPosition(mesh, archTypeOf(fdi), { fdi, actionType });
  const tl = opts.tl || ctx.gsap.timeline();
  const at = opts.at ?? (opts.tl ? tl.duration() : 0);
  const dur = opts.duration ?? (actionType === "overview" ? 1.85 : 1.55);
  lookCam(ctx, framed.targetCameraPosition, framed.targetLookAt, dur, tl, at, {
    keepLocked: opts.keepLocked !== false,
  });
  const surgical = actionType !== "overview";
  setDoF(ctx, null, surgical ? 0.014 : 0.003, {
    aperture: surgical ? 0.00025 : 0.00012,
    duration: dur,
    tl,
    at,
  });
  return { ...framed, fdi, tl };
}

function approachTooth(ctx, fdi, actionType, tl, at = 0) {
  positionCameraForTooth(ctx, fdi, "overview", { tl, at, duration: 1.22, keepLocked: true });
  return positionCameraForTooth(ctx, fdi, actionType, { tl, at: at + 1.22, duration: 1.55, keepLocked: true });
}

function openMouthOn(tl, root, at, duration) {
  const jaw = pick(root, "jawLower_midres", "jawLower");
  if (!jaw) {
    console.warn("[viz] jawLower_midres not found — mouth stays as authored");
    return;
  }
  const pivot = createHinge(jaw);
  if (duration <= 0) {
    pivot.rotation.x = OPEN_ANGLE;
    return;
  }
  tl.to(pivot.rotation, { x: OPEN_ANGLE, duration, ease: EASE }, at);
}

function makeDroplet(scale) {
  const mat = new THREE.MeshPhysicalMaterial({
    color: 0xdbeafe,
    roughness: 0.06,
    metalness: 0.0,
    transmission: 0.94,
    thickness: 0.35,
    transparent: true,
    opacity: 0.58,
    ior: 1.42,
    emissive: new THREE.Color(0x000000),
    emissiveIntensity: 0,
  });
  const mesh = new THREE.Mesh(new THREE.SphereGeometry(scale, 32, 24), mat);
  mesh.name = "prop_sealant_drop";
  mesh.renderOrder = 3;
  return mesh;
}

function makeLED(scale) {
  const g = new THREE.Group();
  g.name = "prop_led_curing";
  const polymer = new THREE.MeshStandardMaterial({ color: 0x111827, roughness: 0.42, metalness: 0.38 });
  const metal = new THREE.MeshStandardMaterial({ color: 0xcbd5e1, roughness: 0.22, metalness: 0.86 });
  const handle = new THREE.Mesh(new THREE.CylinderGeometry(scale * 0.2, scale * 0.26, scale * 2.6, 18), polymer);
  handle.rotation.z = Math.PI / 2.15;
  handle.position.set(-scale * 1.05, scale * 0.15, 0);
  const neck = new THREE.Mesh(new THREE.CylinderGeometry(scale * 0.09, scale * 0.12, scale * 1.4, 12), metal);
  neck.rotation.z = Math.PI / 2.6;
  neck.position.set(scale * 0.15, scale * 0.42, 0);
  const head = new THREE.Mesh(new THREE.CylinderGeometry(scale * 0.28, scale * 0.32, scale * 0.38, 18), polymer);
  head.rotation.z = Math.PI / 2;
  head.position.set(scale * 0.85, scale * 0.62, 0);
  const tip = new THREE.Mesh(
    new THREE.CylinderGeometry(scale * 0.16, scale * 0.16, scale * 0.07, 16),
    new THREE.MeshStandardMaterial({
      color: 0x93c5fd,
      emissive: 0x2563eb,
      emissiveIntensity: 0.12,
      roughness: 0.15,
      metalness: 0.1,
    }),
  );
  tip.rotation.z = Math.PI / 2;
  tip.position.set(scale * 1.06, scale * 0.62, 0);
  g.add(handle, neck, head, tip);
  const light = new THREE.SpotLight(0x93c5fd, 0, scale * 6, Math.PI / 10, 0.4, 2);
  light.position.copy(tip.position);
  const aim = new THREE.Object3D();
  aim.position.set(scale * 1.8, 0, 0);
  g.add(light, aim);
  light.target = aim;
  g.userData.light = light;
  g.userData.tip = tip;
  return g;
}

function steelMat() {
  return new THREE.MeshStandardMaterial({
    color: 0xb7c0c8,
    metalness: 0.86,
    roughness: 0.32,
    envMapIntensity: 0.85,
    emissive: 0x000000,
    emissiveIntensity: 0,
  });
}

function makeHedstromFile(len, rad) {
  const steel = steelMat();
  const g = new THREE.Group();
  g.name = "prop_hedstrom_file";
  const core = new THREE.Mesh(new THREE.CylinderGeometry(rad * 0.2, rad * 0.4, len, 12), steel);
  core.geometry.translate(0, len * 0.5, 0);
  g.add(core);
  const turns = 10;
  for (let k = 0; k < 3; k++) {
    const pts = [];
    const phase = (k * Math.PI * 2) / 3;
    for (let i = 0; i <= 90; i++) {
      const t = i / 90;
      const a = t * Math.PI * 2 * turns + phase;
      const r = rad * (0.26 + 0.28 * t);
      pts.push(new THREE.Vector3(Math.cos(a) * r, t * len, Math.sin(a) * r));
    }
    const blade = new THREE.Mesh(
      new THREE.TubeGeometry(new THREE.CatmullRomCurve3(pts), 90, rad * 0.09, 4, false),
      steel,
    );
    g.add(blade);
  }
  g.traverse((n) => { if (n.isMesh) n.renderOrder = 4; });
  return g;
}

function makeHedstromRig(len, rad, airotorSrc) {
  const rig = new THREE.Group();
  rig.name = "prop_hedstrom_rig";
  const file = makeHedstromFile(len, rad);
  rig.add(file);
  if (airotorSrc) {
    const tool = airotorSrc.clone(true);
    tool.name = "prop_airotor";
    tool.traverse((n) => {
      if (!n.isMesh) return;
      n.castShadow = true;
      n.renderOrder = 4;
      const src = n.material;
      const named = `${src?.name || ""} ${n.name || ""}`.toLowerCase();
      const m = steelMat();
      if (named.includes("tip")) {
        m.color.set(0x9aa3ad);
        m.metalness = 0.88;
        m.roughness = 0.24;
      } else {
        m.color.set(0x3f4754);
        m.metalness = 0.14;
        m.roughness = 0.52;
      }
      n.material = Array.isArray(n.material) ? n.material.map(() => m.clone()) : m;
    });
    let maxDim = 0;
    tool.updateMatrixWorld(true);
    tool.traverse((n) => {
      if (!n.isMesh || !n.geometry) return;
      if (!n.geometry.boundingBox) n.geometry.computeBoundingBox();
      const s0 = n.geometry.boundingBox.getSize(new THREE.Vector3());
      maxDim = Math.max(maxDim, s0.x, s0.y, s0.z);
    });
    if (maxDim < 1e-4) {
      console.warn("[viz] airotor bbox invalid — using fallback scale");
      maxDim = 10;
    }
    tool.scale.setScalar(0.013 / maxDim);
    tool.rotation.z = THREE.MathUtils.degToRad(-78);
    tool.position.set(0, len, 0);
    rig.add(tool);
  } else {
    console.warn("[viz] airotor mesh missing — Hedstrom file plays without handpiece");
    const handle = new THREE.Mesh(
      new THREE.CylinderGeometry(rad * 1.1, rad * 0.9, len * 0.35, 12),
      steelMat(),
    );
    handle.position.y = len + len * 0.12;
    handle.name = "prop_hedstrom_handle";
    rig.add(handle);
  }
  return rig;
}

function burstDebris(ctx, origin, outDir, scale, gsap) {
  const group = new THREE.Group();
  group.name = "prop_hedstrom_debris";
  const n = 12;
  for (let i = 0; i < n; i++) {
    const mat = new THREE.MeshStandardMaterial({
      color: 0x7a3538,
      roughness: 0.86,
      metalness: 0,
      transparent: true,
      opacity: 0.92,
      emissive: 0x000000,
      emissiveIntensity: 0,
    });
    const bit = new THREE.Mesh(new THREE.TetrahedronGeometry(scale * (0.07 + Math.random() * 0.1), 0), mat);
    bit.position.copy(origin);
    bit.position.x += (Math.random() - 0.5) * scale * 0.45;
    bit.position.z += (Math.random() - 0.5) * scale * 0.45;
    bit.scale.setScalar(0.15);
    group.add(bit);
    const dest = origin.clone()
      .addScaledVector(outDir, scale * (1.2 + Math.random() * 2.2))
      .add(new THREE.Vector3((Math.random() - 0.5) * scale, 0, (Math.random() - 0.5) * scale));
    gsap.to(bit.position, { x: dest.x, y: dest.y, z: dest.z, duration: 0.7, ease: "power2.out" });
    gsap.to(bit.scale, { x: 1, y: 1, z: 1, duration: 0.18, ease: "power2.out" });
    gsap.to(mat, { opacity: 0, duration: 0.75, ease: "power2.in", delay: 0.12 });
  }
  ctx.scene.add(group);
  gsap.delayedCall(0.95, () => group.parent?.remove(group));
}

function makeFile(len, rad) {
  return makeHedstromFile(len, rad);
}

function makeGutta(len, rad) {
  const geo = new THREE.CylinderGeometry(rad * 0.92, rad * 0.55, len, 20);
  geo.translate(0, len * 0.5, 0);
  const mesh = new THREE.Mesh(
    geo,
    new THREE.MeshPhysicalMaterial({
      color: 0xf9a8d4,
      roughness: 0.48,
      metalness: 0.05,
      transparent: true,
      opacity: 0,
    }),
  );
  mesh.name = "prop_gutta_percha";
  mesh.renderOrder = 3;
  return mesh;
}

function makeCoreCap(rad) {
  const mesh = new THREE.Mesh(
    new THREE.SphereGeometry(rad, 20, 14, 0, Math.PI * 2, 0, Math.PI * 0.55),
    new THREE.MeshPhysicalMaterial({
      color: 0xd8ccb8,
      roughness: 0.32,
      metalness: 0.02,
      transparent: true,
      opacity: 0,
    }),
  );
  mesh.name = "prop_core_seal";
  mesh.renderOrder = 3;
  return mesh;
}

function attachDirectorMat(mesh, uniforms, key, isSaliva = false) {
  if (!mesh?.isMesh) {
    mesh?.traverse((n) => { if (n.isMesh) attachDirectorMat(n, uniforms, key, isSaliva); });
    return;
  }
  if (!mesh.userData._dirPrev) mesh.userData._dirPrev = mesh.material;
  const m = mesh.material.clone();
  if (key.includes("section") && !isSaliva) m.side = THREE.DoubleSide;
  m.onBeforeCompile = (shader) => {
    shader.uniforms.uPulse = uniforms.uPulse;
    shader.uniforms.uScan = uniforms.uScan;
    shader.uniforms.uSection = uniforms.uSection;
    shader.uniforms.uClean = uniforms.uClean;
    shader.uniforms.uFocus = uniforms.uFocus;
    shader.uniforms.uRadius = uniforms.uRadius;
    shader.uniforms.uPlaneN = uniforms.uPlaneN;
    shader.uniforms.uPlaneC = uniforms.uPlaneC;
    shader.uniforms.uFdiCenter = uniforms.uFdiCenter;
    shader.uniforms.uFdiRadius = uniforms.uFdiRadius;
    shader.uniforms.uExtractMask = uniforms.uExtractMask;
    shader.uniforms.uPrepShrink = uniforms.uPrepShrink;
    shader.uniforms.uCariesSpot = uniforms.uCariesSpot;
    shader.vertexShader = `
      uniform vec3 uFdiCenter; uniform float uFdiRadius; uniform float uExtractMask; uniform float uPrepShrink;
      varying vec3 vWorldPos;
    \n` + shader.vertexShader
      .replace(
        `#include <begin_vertex>`,
        `#include <begin_vertex>
        vec3 wpPre = (modelMatrix * vec4(transformed, 1.0)).xyz;
        float dFdi = length((wpPre - uFdiCenter) * vec3(1.3, 0.6, 1.3));
        float inFdi = 1.0 - smoothstep(uFdiRadius * 0.72, uFdiRadius, dFdi);
        if (uPrepShrink > 0.001 && inFdi > 0.01) {
          transformed -= objectNormal * (uPrepShrink * inFdi * uFdiRadius * 38.0);
        }
        `,
      )
      .replace(
        `#include <worldpos_vertex>`,
        `#include <worldpos_vertex>\nvWorldPos = (modelMatrix * vec4(transformed, 1.0)).xyz;`,
      );
    shader.fragmentShader = `
      uniform float uPulse; uniform float uScan; uniform float uSection; uniform float uClean;
      uniform vec3 uFocus; uniform float uRadius; uniform vec3 uPlaneN; uniform float uPlaneC;
      uniform vec3 uFdiCenter; uniform float uFdiRadius; uniform float uExtractMask; uniform float uPrepShrink; uniform float uCariesSpot;
      varying vec3 vWorldPos;
    ` + shader.fragmentShader.replace(
      `#include <dithering_fragment>`,
      `#include <dithering_fragment>
      vec3 dp = vWorldPos - uFocus;
      float radial = length(dp * vec3(1.35, 0.55, 1.35));
      float inTooth = 1.0 - smoothstep(uRadius * 0.72, uRadius, radial);
      vec3 dpFdi = vWorldPos - uFdiCenter;
      float radialFdi = length(dpFdi * vec3(1.35, 0.55, 1.35));
      float inFdiTooth = 1.0 - smoothstep(uFdiRadius * 0.72, uFdiRadius, radialFdi);
      if (uExtractMask * inFdiTooth > 0.35) discard;
      if (uCariesSpot > 0.01 && inFdiTooth > 0.15 && !${isSaliva ? "true" : "false"}) {
        float cMask = 1.0 - smoothstep(0.0, uFdiRadius * 0.45, radialFdi);
        gl_FragColor.rgb = mix(gl_FragColor.rgb, vec3(0.16, 0.11, 0.08), cMask * uCariesSpot * 0.88);
      }
      if (uScan > 0.001) {
        float ring = smoothstep(0.28, 0.55, radial) * (1.0 - smoothstep(0.70, 0.96, radial));
        float glow = inTooth * ring * uScan * (0.55 + 0.45 * abs(sin(uPulse * 6.28318)));
        gl_FragColor.rgb += vec3(0.10, 0.42, 0.46) * glow * 0.22;
      }
      if (uSection > 0.5 && inTooth > 0.35) {
        float side = dot(vWorldPos, uPlaneN) + uPlaneC;
        if (side > 0.0) discard;
        if (!gl_FrontFacing && !${isSaliva ? "true" : "false"}) {
          float r = clamp(radial / max(uRadius, 0.001), 0.0, 1.0);
          vec3 pulp = mix(vec3(0.70, 0.30, 0.34) * (0.94 + 0.08 * uPulse), vec3(0.85, 0.77, 0.66), uClean);
          vec3 dentin = vec3(0.82, 0.71, 0.55);
          vec3 enamel = vec3(0.88, 0.82, 0.72);
          vec3 col = mix(pulp, dentin, smoothstep(0.12, 0.40, r));
          col = mix(col, enamel, smoothstep(0.55, 0.84, r));
          gl_FragColor = vec4(col, 1.0);
        }
      }`,
    );
  };
  m.customProgramCacheKey = () => `${key}-${isSaliva ? "saliva" : "teeth"}`;
  m.needsUpdate = true;
  mesh.material = m;
}

function detachDirectorMat(mesh) {
  if (!mesh) return;
  mesh.traverse((n) => {
    if (n.isMesh && n.userData._dirPrev) {
      n.material = n.userData._dirPrev;
      delete n.userData._dirPrev;
    }
  });
}

function clearProps(ctx) {
  const kill = [];
  ctx.scene.traverse((o) => {
    if (o.name && o.name.startsWith("prop_")) kill.push(o);
  });
  kill.forEach((o) => o.parent?.remove(o));
}

function hideSaliva(root, hidden) {
  ["SalivaUpper_midres", "SalivaLower_midres", "SalivaUpper", "SalivaLower"].forEach((name) => {
    const o = root.getObjectByName(name);
    if (o) o.visible = !hidden;
  });
}

function toothRadius(arch) {
  const { size } = boxOf(arch);
  return THREE.MathUtils.clamp(Math.max(size.x, size.z) * 0.09, 0.0048, 0.0085);
}

function bindUniforms(molar, radius, planeN) {
  const n = planeN.clone();
  n.y = 0;
  if (n.lengthSq() < 1e-6) n.set(1, 0, 0);
  n.normalize();
  return {
    uPulse: { value: 0 },
    uScan: { value: 0 },
    uSection: { value: 0 },
    uClean: { value: 0 },
    uFocus: { value: molar.clone() },
    uRadius: { value: radius },
    uPlaneN: { value: n },
    uPlaneC: { value: -n.dot(molar) },
    uFdiCenter: { value: molar.clone() },
    uFdiRadius: { value: radius },
    uExtractMask: { value: 0 },
    uPrepShrink: { value: 0 },
    uCariesSpot: { value: 0 },
  };
}

function apexDirOf(molar, upper, canalLen) {
  const orifice = molar.clone();
  const apex = molar.clone().add(new THREE.Vector3(0, (upper ? 1 : -1) * canalLen, 0));
  const dir = apex.clone().sub(orifice).normalize();
  return { orifice, apex, dir };
}

export function playFissureSealant(ctx, fdi = "16") {
  const root = ctx.root;
  const teeth = archMesh(root, fdi);
  const gums = gumsMesh(root, fdi);
  if (!teeth) ctx.caption("Uyarı", "Diş mesh’i bulunamadı; sahne yine de oynar.");

  const target = teeth || root;
  const { upper, right } = fdiMeta(fdi);
  const framed = getOptimalCameraPosition(target, archTypeOf(fdi), { fdi, actionType: "occlusal" });
  const molar = framed.toothPosition;
  const frame = { pos: framed.targetCameraPosition, target: framed.targetLookAt, dist: framed.dist };
  const radius = toothRadius(target);
  const uniforms = bindUniforms(molar, radius, frame.pos.clone().sub(molar));
  const { dir: apexDir } = apexDirOf(molar, upper, radius * 3.2);
  const occlusalOut = apexDir.clone().negate();

  ctx.restore();
  clearProps(ctx);
  attachDirectorMat(target, uniforms, "sealant-scan-v2");

  const drop = makeDroplet(radius * 0.16);
  drop.position.copy(molar).addScaledVector(occlusalOut, radius * 4.2);
  drop.scale.setScalar(0.01);
  drop.visible = false;
  ctx.scene.add(drop);

  const led = makeLED(radius * 1.65);
  led.visible = false;
  ctx.scene.add(led);

  const tl = ctx.gsap.timeline({ paused: true, defaults: { ease: EASE } });

  const step1 = () => {
    const at = tl.duration();
    tl.add(() => {
      ctx.caption("Hamle 1 — Kadraj", archApproachLabel(fdi));
    }, at);
    approachTooth(ctx, fdi, "occlusal", tl, at);
  };

  const step2 = () => {
    const at = tl.duration();
    tl.add(() => ctx.caption("Hamle 2 — Odak & temizlik", "Fissür hatları hafif turkuaz konturla işaretleniyor"), at);
    tl.to(uniforms.uScan, { value: 1, duration: 0.45 }, at);
    tl.to(uniforms.uPulse, { value: 1, duration: 1.55, ease: "sine.inOut" }, at);
    tl.to(uniforms.uScan, { value: 0.18, duration: 0.4 }, at + 1.55);
  };

  const step3 = () => {
    const at = tl.duration();
    const land = molar.clone().addScaledVector(occlusalOut, radius * 0.12);
    tl.add(() => {
      ctx.caption("Hamle 3 — Uygulama", "Akışkan örtücü oluklara iniyor ve yayılıyor");
      drop.visible = true;
    }, at);
    tl.to(drop.scale, { x: 1, y: 1, z: 1, duration: 0.55 }, at);
    tl.to(drop.position, { x: land.x, y: land.y, z: land.z, duration: 1.15 }, at);
    tl.to(drop.scale, { x: 1.75, y: 0.18, z: 1.45, duration: 0.95, ease: "power2.inOut" });
    tl.to(drop.material.color, { r: 0.94, g: 0.90, b: 0.84, duration: 0.8 }, "<");
    tl.to(drop.material, { transmission: 0.38, opacity: 0.88, roughness: 0.22, duration: 0.8 }, "<");
  };

  const step4 = () => {
    const at = tl.duration();
    const hover = molar.clone()
      .addScaledVector(occlusalOut, 0.002)
      .add(new THREE.Vector3(right ? -radius * 0.9 : radius * 0.9, 0, radius * 0.35));
    const park = hover.clone().add(new THREE.Vector3(frame.pos.x - molar.x, occlusalOut.y, 0.03).setLength(radius * 8));
    led.position.copy(park);
    led.lookAt(molar);
    tl.add(() => {
      ctx.caption("Hamle 4 — Polimerizasyon", "LED uç 2 mm yukarıda, mavi fototerapi");
      led.visible = true;
    }, at);
    tl.to(led.position, {
      x: hover.x, y: hover.y, z: hover.z, duration: 0.9,
      onUpdate: () => led.lookAt(molar),
    }, at);
    tl.to(led.userData.light, { intensity: 0.85, duration: 0.32 });
    tl.to(led.userData.tip.material, { emissiveIntensity: 0.7, duration: 0.32 }, "<");
    tl.to(drop.material, { transmission: 0.02, opacity: 1, roughness: 0.32, metalness: 0.0, duration: 0.5 }, "<0.08");
    tl.to(drop.material.color, { r: 0.95, g: 0.91, b: 0.85, duration: 0.5 }, "<");
    tl.to(led.userData.light, { intensity: 0, duration: 0.45 }, "+=0.2");
    tl.to(led.userData.tip.material, { emissiveIntensity: 0.15, duration: 0.45 }, "<");
    tl.add(() => {
      eachMat(gums, (m) => { if (m.emissive) m.emissiveIntensity = 0; });
      detachDirectorMat(target);
      ctx.restore();
      ensureOpenMouth(root, ctx.gsap, OPEN_ANGLE, 0);
      ctx.caption("Tamam", "Fissür örtücü dişle bütünleşti");
      setDoF(ctx, frame.dist, 0.003, { aperture: 0.00012 });
      ctx.controls.enabled = true;
    });
  };

  step1();
  tl.add(() => {}, "+=0.32");
  step2();
  tl.add(() => {}, "+=0.08");
  step3();
  tl.add(() => {}, "+=0.08");
  step4();
  console.info("[viz] sealant frame", { molar: molar.toArray(), cam: frame.pos.toArray(), dist: frame.dist });
  requestAnimationFrame(() => tl.play(0));
  return tl;
}

export function playHedstromCleaningAnimation(ctx, fdi = "16", opts = {}) {
  const FILE_EASE = "power2.inOut";
  const root = ctx.root;
  const { upper } = fdiMeta(fdi);
  const teeth = archMesh(root, fdi);
  if (!teeth) {
    console.warn("[viz] Hedstrom: arch mesh not found for FDI", fdi);
    ctx.caption("Uyarı", "Diş mesh’i bulunamadı; Hedstrom sahnesi yine de oynar.");
  }
  const target = teeth || root;
  const framed = getOptimalCameraPosition(target, archTypeOf(fdi), { fdi, actionType: "section" });
  const molar = framed.toothPosition;
  const frame = { pos: framed.targetCameraPosition, target: framed.targetLookAt, dist: framed.dist };
  const radius = toothRadius(target);
  const canalLen = radius * 3.4;
  const { orifice, apex, dir: apexDir } = apexDirOf(molar, upper, canalLen);
  const uniforms = bindUniforms(molar, radius * 1.05, frame.pos.clone().sub(molar));

  if (!opts.skipRestore) {
    ctx.restore();
    clearProps(ctx);
    ctx.renderer.localClippingEnabled = false;
    attachDirectorMat(target, uniforms, "endo-hedstrom-section-v1");
  }

  const outDir = apexDir.clone().negate();
  const airotorSrc = ctx.airotor?.scene || ctx.airotor;
  if (!airotorSrc) console.warn("[viz] airotor.glb not loaded — Hedstrom plays without handpiece");
  const rig = makeHedstromRig(canalLen * 1.08, radius * 0.085, airotorSrc);
  rig.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), outDir);
  const parked = orifice.clone().addScaledVector(outDir, canalLen * 1.35);
  const working = apex.clone().addScaledVector(outDir, radius * 0.08);
  rig.position.copy(parked);
  rig.visible = false;
  ctx.scene.add(rig);

  const tl = opts.tl || ctx.gsap.timeline({ paused: true, defaults: { ease: FILE_EASE } });

  const step1 = () => {
    const at = tl.duration();
    tl.add(() => {
      ctx.caption("Hamle 1 — Kesit", `${archApproachLabel(fdi)} · mine / dentin / pulpa`);
    }, at);
    approachTooth(ctx, fdi, "section", tl, at);
    tl.to(uniforms.uSection, { value: 1, duration: 0.7, ease: FILE_EASE }, at + 2.15);
    tl.to(uniforms.uPulse, { value: 1, duration: 0.85, ease: "sine.inOut" }, at + 2.35);
  };

  const step2 = () => {
    const at = tl.duration();
    tl.add(() => {
      ctx.caption("Hamle 2 — Hedstrom girişi", "Spiral eğe kanal ekseninde apekse iniyor");
      rig.visible = true;
    }, at);
    tl.to(rig.position, { x: working.x, y: working.y, z: working.z, duration: 1.2, ease: FILE_EASE }, at);
  };

  const step3 = () => {
    const at = tl.duration();
    tl.add(() => ctx.caption("Hamle 3 — Pull-stroke", "Doğrusal çekme vuruşu · pulpa ve debris temizleniyor"), at);
    const strokes = 5;
    for (let i = 0; i < strokes; i++) {
      const pushTo = orifice.clone().lerp(working, 0.94 - i * 0.05);
      const pullTo = orifice.clone().lerp(working, 0.4);
      tl.to(rig.position, { x: pushTo.x, y: pushTo.y, z: pushTo.z, duration: 0.22, ease: "power2.in" });
      tl.to(rig.position, {
        x: pullTo.x, y: pullTo.y, z: pullTo.z, duration: 0.42, ease: "power2.out",
        onStart: () => burstDebris(ctx, rig.position.clone(), outDir, radius, ctx.gsap),
      });
      tl.to(uniforms.uClean, { value: (i + 1) / strokes, duration: 0.42, ease: FILE_EASE }, "<");
      tl.to(uniforms.uPulse, { value: 1 - (i + 1) / strokes, duration: 0.42, ease: FILE_EASE }, "<");
    }
  };

  const step4 = () => {
    const at = tl.duration();
    tl.add(() => ctx.caption("Hamle 4 — Çıkış", "Hedstrom geri çekildi · steril kanal obturasyona hazır"), at);
    tl.to(rig.position, { x: working.x, y: working.y, z: working.z, duration: 0.28, ease: "power2.in" }, at);
    tl.to(rig.position, { x: parked.x, y: parked.y, z: parked.z, duration: 0.9, ease: FILE_EASE });
    tl.add(() => { rig.visible = false; });
  };

  step1();
  tl.add(() => {}, "+=0.28");
  step2();
  tl.add(() => {}, "+=0.12");
  step3();
  tl.add(() => {}, "+=0.12");
  step4();

  if (opts.autoplay !== false && !opts.tl) requestAnimationFrame(() => tl.play(0));
  console.info("[viz] hedstrom frame", { molar: molar.toArray(), cam: frame.pos.toArray(), dist: frame.dist, upper });
  return { tl, uniforms, target, molar, frame, orifice, apex, apexDir, radius, canalLen, rig };
}

export function playRootCanal(ctx, fdi = "16") {
  const root = ctx.root;
  const { uniforms, target, frame, orifice, apex, apexDir, radius, canalLen, tl } =
    playHedstromCleaningAnimation(ctx, fdi, { autoplay: false });

  const gutta = makeGutta(canalLen, radius * 0.16);
  gutta.position.copy(apex);
  gutta.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), orifice.clone().sub(apex).normalize());
  gutta.scale.set(1, 0.001, 1);
  ctx.scene.add(gutta);

  const cap = makeCoreCap(radius * 0.85);
  cap.position.copy(orifice);
  cap.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), apexDir.clone().negate());
  cap.visible = false;
  ctx.scene.add(cap);

  tl.add(() => {}, "+=0.18");
  const at = tl.duration();
  tl.add(() => {
    ctx.caption("Hamle 5 — Guta-perka", "Obtürasyon, kor örtü ve doğal forma dönüş");
    cap.visible = true;
  }, at);
  tl.to(gutta.scale, { y: 1, duration: 1.2, ease: "power2.out" }, at);
  tl.to(gutta.material, { opacity: 0.96, duration: 0.85 }, at);
  tl.to(gutta.material.color, { r: 0.98, g: 0.74, b: 0.62, duration: 0.7 }, at + 0.35);
  tl.to(cap.material, { opacity: 1, duration: 0.55 }, at + 0.85);
  tl.to(uniforms.uSection, { value: 0, duration: 0.85 }, at + 1.15);
  tl.add(() => {
    detachDirectorMat(target);
    ctx.restore();
    ensureOpenMouth(root, ctx.gsap, OPEN_ANGLE, 0);
  });
  tl.add(() => {
    gutta.visible = false;
    cap.visible = false;
    ctx.caption("Tamam", "Kanal mühürlendi");
    setDoF(ctx, frame.dist, 0.003, { aperture: 0.00012 });
    ctx.controls.enabled = true;
  });

  requestAnimationFrame(() => tl.play(0));
  return tl;
}

export { lookCam as flyCamera };

export const Cinematics = {
  fissure_sealant: playFissureSealant,
  root_canal: playRootCanal,
  hedstrom_cleaning: (ctx, fdi) => playHedstromCleaningAnimation(ctx, fdi).tl,
};
