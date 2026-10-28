import * as THREE from "three";

const EASE = "cubic-bezier(0.22, 1, 0.36, 1)";
const OPEN_ANGLE = 0.16;
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
  return {
    n,
    upper: n < 30,
    right: (n >= 11 && n <= 18) || (n >= 41 && n <= 48),
    distal: Math.max(0, Math.min(1, ((n % 10) - 1) / 7)),
  };
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

function lookCam(ctx, pos, target, dur, tl, at) {
  const t = at ?? tl.duration();
  tl.add(() => { ctx.controls.enabled = false; }, t);
  tl.to(ctx.camera.position, { x: pos.x, y: pos.y, z: pos.z, duration: dur, ease: EASE }, t);
  tl.to(ctx.controls.target, { x: target.x, y: target.y, z: target.z, duration: dur, ease: EASE }, t);
  tl.add(() => { ctx.controls.enabled = true; }, t + dur);
}

function setDoF(ctx, focus, blur) {
  if (!ctx.bokeh?.uniforms) return;
  const u = ctx.bokeh.uniforms;
  if (u.focus) ctx.gsap.to(u.focus, { value: focus, duration: 1.05, ease: EASE });
  if (u.maxblur) ctx.gsap.to(u.maxblur, { value: blur, duration: 1.05, ease: EASE });
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

function surgicalFrame(root, molar, fdi, mode = "occlusal") {
  const { upper, right } = fdiMeta(fdi);
  const buccalX = right ? -1 : 1;
  const occlusalY = upper ? 1 : -1;
  const elev = THREE.MathUtils.degToRad(mode === "section" ? 32 : 40);
  const yaw = THREE.MathUtils.degToRad(34);
  const dir = new THREE.Vector3(
    buccalX * Math.cos(elev) * 0.88,
    occlusalY * Math.sin(elev),
    Math.cos(elev) * Math.sin(yaw),
  ).normalize();
  const dist = mode === "section" ? 0.074 : 0.07;
  return { pos: molar.clone().addScaledVector(dir, dist), target: molar.clone(), dist };
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

function makeFile(len, rad) {
  const pts = [];
  for (let i = 0; i <= 110; i++) {
    const t = i / 110;
    pts.push(new THREE.Vector3(
      Math.cos(t * Math.PI * 14) * rad,
      -t * len,
      Math.sin(t * Math.PI * 14) * rad,
    ));
  }
  const mesh = new THREE.Mesh(
    new THREE.TubeGeometry(new THREE.CatmullRomCurve3(pts), 110, rad * 0.5, 8, false),
    new THREE.MeshStandardMaterial({ color: 0xe2e8f0, metalness: 0.94, roughness: 0.16 }),
  );
  mesh.name = "prop_endo_file";
  mesh.renderOrder = 4;
  return mesh;
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
      color: 0xf8f1e7,
      roughness: 0.28,
      metalness: 0.04,
      transparent: true,
      opacity: 0,
    }),
  );
  mesh.name = "prop_core_seal";
  mesh.renderOrder = 3;
  return mesh;
}

function attachDirectorMat(mesh, uniforms, key) {
  if (!mesh?.isMesh) {
    mesh?.traverse((n) => { if (n.isMesh) attachDirectorMat(n, uniforms, key); });
    return;
  }
  if (!mesh.userData._dirPrev) mesh.userData._dirPrev = mesh.material;
  const m = mesh.material.clone();
  if (key.includes("section")) m.side = THREE.DoubleSide;
  m.onBeforeCompile = (shader) => {
    shader.uniforms.uPulse = uniforms.uPulse;
    shader.uniforms.uScan = uniforms.uScan;
    shader.uniforms.uSection = uniforms.uSection;
    shader.uniforms.uClean = uniforms.uClean;
    shader.uniforms.uFocus = uniforms.uFocus;
    shader.uniforms.uRadius = uniforms.uRadius;
    shader.uniforms.uPlaneN = uniforms.uPlaneN;
    shader.uniforms.uPlaneC = uniforms.uPlaneC;
    shader.vertexShader = `varying vec3 vWorldPos;\n` + shader.vertexShader.replace(
      `#include <worldpos_vertex>`,
      `#include <worldpos_vertex>\nvWorldPos = (modelMatrix * vec4(transformed, 1.0)).xyz;`,
    );
    shader.fragmentShader = `
      uniform float uPulse; uniform float uScan; uniform float uSection; uniform float uClean;
      uniform vec3 uFocus; uniform float uRadius; uniform vec3 uPlaneN; uniform float uPlaneC;
      varying vec3 vWorldPos;
    ` + shader.fragmentShader.replace(
      `#include <dithering_fragment>`,
      `#include <dithering_fragment>
      vec3 dp = vWorldPos - uFocus;
      float radial = length(dp * vec3(1.35, 0.55, 1.35));
      float inTooth = 1.0 - smoothstep(uRadius * 0.72, uRadius, radial);
      if (uScan > 0.001) {
        float ring = smoothstep(0.28, 0.55, radial) * (1.0 - smoothstep(0.70, 0.96, radial));
        float glow = inTooth * ring * uScan * (0.55 + 0.45 * abs(sin(uPulse * 6.28318)));
        gl_FragColor.rgb += vec3(0.10, 0.42, 0.46) * glow * 0.22;
      }
      if (uSection > 0.5 && inTooth > 0.35) {
        float side = dot(vWorldPos, uPlaneN) + uPlaneC;
        if (side > 0.0) discard;
        if (!gl_FrontFacing) {
          float r = clamp(radial / max(uRadius, 0.001), 0.0, 1.0);
          vec3 pulp = mix(vec3(0.70, 0.30, 0.34) * (0.94 + 0.08 * uPulse), vec3(0.14, 0.12, 0.12), uClean);
          vec3 dentin = vec3(0.82, 0.70, 0.50);
          vec3 enamel = vec3(0.95, 0.92, 0.87);
          vec3 col = mix(pulp, dentin, smoothstep(0.12, 0.40, r));
          col = mix(col, enamel, smoothstep(0.55, 0.84, r));
          gl_FragColor = vec4(col, 1.0);
        }
      }`,
    );
  };
  m.customProgramCacheKey = () => key;
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
  const teeth = pick(root, "Tooth_Upper_Molar_16", "tooth_16", "TeethUpper_midres", "TeethUpper");
  const gums = pick(root, "GumsUpper_midres", "GumsUpper");
  if (!teeth) ctx.caption("Uyarı", "Üst çene diş mesh’i bulunamadı; sahne yine de oynar.");

  const target = teeth || root;
  const { upper, right } = fdiMeta(fdi);
  const molar = fdiPoint(target, fdi);
  const frame = surgicalFrame(root, molar, fdi, "occlusal");
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
      ctx.caption("Hamle 1 — Kadraj", `Diş ${fdi} okluzal makro, 40° çapraz bakış`);
      setDoF(ctx, frame.dist, 0.005);
    }, at);
    openMouthOn(tl, root, at, 1.05);
    lookCam(ctx, frame.pos, molar, 1.7, tl, at + 0.12);
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
      setDoF(ctx, frame.dist * 3.2, 0.002);
    });
  };

  step1();
  tl.add(() => {}, "+=0.05");
  step2();
  tl.add(() => {}, "+=0.05");
  step3();
  tl.add(() => {}, "+=0.05");
  step4();
  console.info("[viz] sealant frame", { molar: molar.toArray(), cam: frame.pos.toArray(), dist: frame.dist });
  requestAnimationFrame(() => tl.play(0));
  return tl;
}

export function playRootCanal(ctx, fdi = "16") {
  const root = ctx.root;
  const { upper } = fdiMeta(fdi);
  const teeth = pick(
    root,
    upper ? "Tooth_Upper_Molar_16" : "Tooth_Lower_Molar_36",
    upper ? "TeethUpper_midres" : "TeethLower_midres",
    "TeethUpper_midres",
    "TeethUpper",
  );
  const gums = pick(root, upper ? "GumsUpper_midres" : "GumsLower_midres", "GumsUpper", "GumsLower");
  const target = teeth || root;
  if (!teeth) ctx.caption("Uyarı", "Kesit için diş mesh’i yok; shader yine de denenir.");

  const molar = fdiPoint(target, fdi);
  const frame = surgicalFrame(root, molar, fdi, "section");
  const radius = toothRadius(target);
  const canalLen = radius * 3.4;
  const { orifice, apex, dir: apexDir } = apexDirOf(molar, upper, canalLen);
  const uniforms = bindUniforms(molar, radius * 1.05, frame.pos.clone().sub(molar));

  ctx.restore();
  clearProps(ctx);
  ctx.renderer.localClippingEnabled = false;
  attachDirectorMat(target, uniforms, "endo-section-v2");

  const file = makeFile(canalLen * 1.15, radius * 0.07);
  file.quaternion.setFromUnitVectors(new THREE.Vector3(0, -1, 0), apexDir);
  file.position.copy(orifice).addScaledVector(apexDir, -canalLen * 1.2);
  file.visible = false;
  ctx.scene.add(file);

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

  const tl = ctx.gsap.timeline({ paused: true, defaults: { ease: EASE } });

  const step1 = () => {
    const at = tl.duration();
    tl.add(() => {
      ctx.caption("Hamle 1 — Kesit", `Diş ${fdi} dikey kesit: mine / dentin / pulpa`);
      setDoF(ctx, frame.dist, 0.005);
    }, at);
    openMouthOn(tl, root, at, 1.05);
    lookCam(ctx, frame.pos, molar, 1.65, tl, at + 0.12);
    tl.to(uniforms.uSection, { value: 1, duration: 0.7 }, at + 0.45);
  };

  const step2 = () => {
    const at = tl.duration();
    tl.add(() => ctx.caption("Hamle 2 — Enfeksiyon", "Pulpa odasında hafif inflamasyon vurgusu"), at);
    tl.to(uniforms.uPulse, { value: 1, duration: 0.55, yoyo: true, repeat: 3, ease: "sine.inOut" }, at);
  };

  const step3 = () => {
    const at = tl.duration();
    const start = file.position.clone();
    const depth = apex.clone().addScaledVector(apexDir, -radius * 0.12);
    tl.add(() => {
      ctx.caption("Hamle 3 — Mekanik temizlik", "Endodontik eğe kanalda dönerek iniyor");
      file.visible = true;
    }, at);
    tl.to(file.position, { x: orifice.x, y: orifice.y, z: orifice.z, duration: 0.55 }, at);
    tl.to(file.rotation, { y: Math.PI * 10, duration: 2.05, ease: "none" }, at);
    tl.to(file.position, { x: depth.x, y: depth.y, z: depth.z, duration: 1.7, ease: "power1.inOut" }, at + 0.15);
    tl.to(uniforms.uClean, { value: 1, duration: 1.35 }, at + 0.35);
    tl.to(uniforms.uPulse, { value: 0, duration: 1.0 }, at + 0.35);
    tl.to(file.position, { x: start.x, y: start.y, z: start.z, duration: 0.7, ease: "power2.in" });
    tl.add(() => { file.visible = false; });
  };

  const step4 = () => {
    const at = tl.duration();
    tl.add(() => {
      ctx.caption("Hamle 4 — Guta-perka", "Obtürasyon, kor örtü ve doğal forma dönüş");
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
      setDoF(ctx, frame.dist * 3.4, 0.002);
    });
  };

  step1();
  tl.add(() => {}, "+=0.05");
  step2();
  tl.add(() => {}, "+=0.05");
  step3();
  tl.add(() => {}, "+=0.05");
  step4();
  console.info("[viz] endo frame", { molar: molar.toArray(), cam: frame.pos.toArray(), dist: frame.dist });
  requestAnimationFrame(() => tl.play(0));
  return tl;
}

export const Cinematics = {
  fissure_sealant: playFissureSealant,
  root_canal: playRootCanal,
};
