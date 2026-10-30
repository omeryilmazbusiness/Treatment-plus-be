import * as THREE from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import { GLTFLoader } from "three/addons/loaders/GLTFLoader.js";
import { RoomEnvironment } from "three/addons/environments/RoomEnvironment.js";
import { EffectComposer } from "three/addons/postprocessing/EffectComposer.js";
import { RenderPass } from "three/addons/postprocessing/RenderPass.js";
import { BokehPass } from "three/addons/postprocessing/BokehPass.js";
import { UnrealBloomPass } from "three/addons/postprocessing/UnrealBloomPass.js";
import { OutputPass } from "three/addons/postprocessing/OutputPass.js";
import gsap from "gsap";
import { Cinematics, playFissureSealant, playRootCanal, ensureOpenMouth } from "./cinematics.js?v=32";

const MODEL_URL = new URL("./mouth.glb", window.location.href).toString();

function inspectModel(root) {
  const rows = [];
  root.traverse((o) => {
    rows.push({
      name: o.name || "(unnamed)",
      type: o.type,
      mesh: !!o.isMesh,
      verts: o.isMesh ? o.geometry?.attributes?.position?.count ?? 0 : 0,
    });
  });
  console.group("GLTF mesh / node graph");
  console.table(rows);
  console.groupEnd();
  const el = document.getElementById("graph");
  if (el) {
    el.textContent = rows
      .filter((r) => r.name !== "(unnamed)" || r.mesh)
      .map((r) => `${r.mesh ? "MESH" : r.type.padEnd(10)}  ${r.name}${r.verts ? `  v=${r.verts}` : ""}`)
      .join("\n");
  }
}

function invertMap(tex) {
  if (!tex?.image) return tex;
  const img = tex.image;
  const w = img.width || img.videoWidth || 0;
  const h = img.height || img.videoHeight || 0;
  if (!w || !h) return tex;
  const c = document.createElement("canvas");
  c.width = w;
  c.height = h;
  const ctx = c.getContext("2d");
  ctx.drawImage(img, 0, 0);
  const data = ctx.getImageData(0, 0, w, h);
  for (let i = 0; i < data.data.length; i += 4) {
    data.data[i] = 255 - data.data[i];
    data.data[i + 1] = 255 - data.data[i + 1];
    data.data[i + 2] = 255 - data.data[i + 2];
  }
  ctx.putImageData(data, 0, 0);
  const out = tex.clone();
  out.image = c;
  out.needsUpdate = true;
  return out;
}

function restoreBlendGlass(src) {
  const phys = new THREE.MeshPhysicalMaterial();
  if (src) phys.copy(src);
  phys.name = src?.name || "saliva";
  phys.color.setRGB(0.86, 0.94, 1);
  phys.metalness = 0;
  phys.roughness = 0.03;
  phys.ior = 1.33;
  phys.transmission = 1;
  phys.thickness = 0.007;
  phys.attenuationColor = new THREE.Color(0xdbeafe);
  phys.attenuationDistance = 0.04;
  phys.transparent = true;
  phys.opacity = 1;
  phys.depthWrite = false;
  phys.side = THREE.FrontSide;
  phys.polygonOffset = true;
  phys.polygonOffsetFactor = -2;
  phys.polygonOffsetUnits = -2;
  if (src?.normalMap) {
    phys.normalMap = src.normalMap;
    phys.normalScale = new THREE.Vector2(0.55, 0.55);
  }
  const mask = src?.alphaMap || src?.map;
  phys.map = null;
  phys.alphaMap = mask || null;
  phys.envMapIntensity = 1.7;
  return phys;
}

function useAuthoredMaterials(root) {
  root.traverse((n) => {
    if (!n.isMesh) return;
    n.castShadow = true;
    n.receiveShadow = true;
    const srcList = Array.isArray(n.material) ? n.material : [n.material];
    const next = srcList.map((src) => {
      if (!src) return src;
      const name = `${src.name || ""} ${n.name || ""}`.toLowerCase();
      if (name.includes("saliva")) {
        n.renderOrder = 3;
        return restoreBlendGlass(src);
      }
      const phys = src.isMeshPhysicalMaterial ? src : new THREE.MeshPhysicalMaterial();
      if (phys !== src) phys.copy(src);
      phys.name = src.name;
      phys.metalness = 0;
      phys.transmission = 0;
      phys.thickness = 0;
      phys.transparent = false;
      phys.opacity = 1;
      phys.side = /gums/i.test(n.name) ? THREE.DoubleSide : THREE.FrontSide;
      phys.ior = src.ior || 1.45;
      phys.specularIntensity = src.specularIntensity ?? 0.64;
      phys.clearcoat = 0.42;
      phys.clearcoatRoughness = 0.06;
      phys.sheen = 0.18;
      phys.sheenColor = new THREE.Color(0xffd8c4);
      phys.envMapIntensity = 1.25;
      if (src.roughnessMap && !src.userData.glossInverted) {
        phys.roughnessMap = invertMap(src.roughnessMap);
        phys.roughness = 1;
        src.userData.glossInverted = true;
      }
      return phys;
    });
    n.material = Array.isArray(n.material) ? next : next[0];
  });
}

function snapshotMats(obj) {
  obj.traverse((n) => {
    if (!n.isMesh) return;
    const mats = Array.isArray(n.material) ? n.material.map((m) => m.clone()) : n.material.clone();
    n.material = mats;
    const list = Array.isArray(n.material) ? n.material : [n.material];
    n.userData.base = list.map((m) => ({
      color: m.color?.clone(),
      emissive: m.emissive?.clone(),
      roughness: m.roughness,
      metalness: m.metalness,
      opacity: m.opacity,
      transparent: m.transparent,
      emissiveIntensity: m.emissiveIntensity,
    }));
    n.userData.basePos = n.position.clone();
    n.userData.baseScale = n.scale.clone();
    n.userData.baseQuat = n.quaternion.clone();
    n.userData.baseVisible = n.visible;
    n.userData.baseMat = n.material;
  });
}

function restoreMats(obj) {
  if (!obj) return;
  obj.traverse((n) => {
    if (!n.isMesh || !n.userData.base) return;
    if (n.userData.baseMat) n.material = n.userData.baseMat;
    const list = Array.isArray(n.material) ? n.material : [n.material];
    list.forEach((m, i) => {
      const b = n.userData.base[i];
      if (!b || !m) return;
      if (b.color && m.color) m.color.copy(b.color);
      if (b.emissive && m.emissive) m.emissive.copy(b.emissive);
      if (m.roughness != null) m.roughness = b.roughness;
      if (m.metalness != null) m.metalness = b.metalness;
      m.opacity = b.opacity;
      m.transparent = b.transparent;
      if (m.emissiveIntensity != null) m.emissiveIntensity = b.emissiveIntensity ?? 0;
      m.clippingPlanes = [];
      m.needsUpdate = true;
    });
    if (n.userData.basePos) n.position.copy(n.userData.basePos);
    if (n.userData.baseScale) n.scale.copy(n.userData.baseScale);
    if (n.userData.baseQuat) n.quaternion.copy(n.userData.baseQuat);
    n.visible = n.userData.baseVisible !== false;
  });
}

class Director {
  constructor() {
    this.scene = new THREE.Scene();
    this.scene.background = new THREE.Color().setRGB(0.050876, 0.050876, 0.050876);
    this.camera = new THREE.PerspectiveCamera(39.6, 1, 0.1, 100);
    this.renderer = new THREE.WebGLRenderer({ antialias: true, powerPreference: "high-performance" });
    this.renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
    this.renderer.outputColorSpace = THREE.SRGBColorSpace;
    this.renderer.toneMapping = THREE.ACESFilmicToneMapping;
    this.renderer.toneMappingExposure = 1.0;
    this.renderer.localClippingEnabled = false;
    this.renderer.shadowMap.enabled = true;
    this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
    document.getElementById("stage").appendChild(this.renderer.domElement);
    this.controls = new OrbitControls(this.camera, this.renderer.domElement);
    this.controls.enableDamping = true;
    this.pmrem = new THREE.PMREMGenerator(this.renderer);
    this.composer = new EffectComposer(this.renderer);
    this.composer.addPass(new RenderPass(this.scene, this.camera));
    try {
      this.bokeh = new BokehPass(this.scene, this.camera, { focus: 1.4, aperture: 0.00012, maxblur: 0.003 });
      this.composer.addPass(this.bokeh);
    } catch (err) {
      console.warn("[viz] DoF unavailable", err);
      this.bokeh = null;
    }
    this.composer.addPass(new UnrealBloomPass(new THREE.Vector2(innerWidth, innerHeight), 0.16, 0.42, 0.86));
    this.composer.addPass(new OutputPass());
    this.root = null;
    this.playlist = { clips: [] };
    this.clipIndex = 0;
    this.activeTL = null;
    window.addEventListener("resize", () => this.resize());
    this.resize();
    this.camera.position.set(0.2, 0.18, 0.55);
    this.loop();
  }

  resize() {
    const w = innerWidth;
    const h = innerHeight;
    this.camera.aspect = w / h;
    this.camera.updateProjectionMatrix();
    this.renderer.setSize(w, h);
    this.composer.setSize(w, h);
  }

  loop() {
    this.controls.update();
    this.composer.render();
    requestAnimationFrame(() => this.loop());
  }

  caption(title, body) {
    document.getElementById("caption").innerHTML = `<b>${title}</b><span>${body || ""}</span>`;
  }

  ctx() {
    return {
      THREE,
      gsap,
      scene: this.scene,
      camera: this.camera,
      controls: this.controls,
      renderer: this.renderer,
      bokeh: this.bokeh,
      root: this.root,
      caption: (t, b) => this.caption(t, b),
      restore: () => {
        restoreMats(this.root);
        this.renderer.localClippingEnabled = false;
      },
    };
  }

  async loadModel() {
    this.caption("Model", "GLTF ağız modeli yükleniyor…");
    const gltf = await new GLTFLoader().loadAsync(MODEL_URL);
    this.root = gltf.scene;
    useAuthoredMaterials(this.root);
    snapshotMats(this.root);
    this.scene.add(this.root);
    const authoredLights = [];
    this.root.traverse((o) => {
      if (o.isLight) {
        o.castShadow = true;
        if (o.shadow) o.shadow.mapSize.set(2048, 2048);
        // Blender punctual candela is correct at the scene camera (~10u).
        // Clinical framing is much closer; keep the same illuminance.
        if (o.intensity > 100) o.intensity *= 0.0032;
        authoredLights.push(o);
      }
    });
    if (!authoredLights.length) {
      const key = new THREE.PointLight(0xffffff, 1000, 0, 2);
      key.name = "Light";
      key.position.set(4.076, 5.904, -1.005);
      key.castShadow = true;
      key.shadow.mapSize.set(2048, 2048);
      this.root.add(key);
    }
    this.scene.environment = this.pmrem.fromScene(new RoomEnvironment(), 0.04).texture;
    this.scene.environmentIntensity = 0.82;
    const salivaMeshes = [];
    this.root.traverse((n) => {
      if (n.isMesh && /saliva/i.test(n.name)) salivaMeshes.push(n);
    });
    salivaMeshes.forEach((n) => { n.visible = false; });
    const mouthEnv = this.pmrem.fromScene(this.scene, 0.03).texture;
    salivaMeshes.forEach((n) => {
      n.visible = true;
      const mats = Array.isArray(n.material) ? n.material : [n.material];
      mats.forEach((m) => {
        m.envMap = mouthEnv;
        m.envMapIntensity = 1.7;
        m.needsUpdate = true;
      });
    });
    inspectModel(this.root);
    ensureOpenMouth(this.root, gsap, 0.12, 0);
    window.__director = this;
    const focus = this.root.getObjectByName("TeethTongueSet_midres")
      || this.root.getObjectByName("TeethUpper_midres")
      || this.root;
    const box = new THREE.Box3().setFromObject(focus);
    const c = box.getCenter(new THREE.Vector3());
    const size = box.getSize(new THREE.Vector3());
    const dist = Math.max(size.length() * 1.55, 0.28);
    const blendCam = (gltf.cameras && gltf.cameras[0]) || null;
    if (blendCam) {
      this.camera.fov = blendCam.fov;
      this.camera.near = Math.min(blendCam.near, 0.01);
      this.camera.far = blendCam.far;
      this.camera.updateProjectionMatrix();
    }
    this.camera.position.set(c.x + dist * 0.42, c.y + dist * 0.18, c.z + dist * 0.9);
    this.controls.target.copy(c);
    this.camera.lookAt(c);
    this.caption("Hazır", "Blend sahnesi: ışık, kamera, tükürük ve midres mesh.");
  }

  kill() {
    this.activeTL?.kill();
    this.activeTL = null;
    restoreMats(this.root);
    this.renderer.localClippingEnabled = false;
    const drop = [];
    this.scene.traverse((o) => { if (o.name?.startsWith("prop_")) drop.push(o); });
    drop.forEach((o) => o.parent?.remove(o));
  }

  playClip(i) {
    this.clipIndex = i;
    const clip = this.playlist.clips[i];
    if (!clip || !this.root) return;
    const fn = Cinematics[clip.code] || (clip.code === "fissure_sealant" ? playFissureSealant : clip.code === "root_canal" ? playRootCanal : null);
    if (!fn) {
      this.caption("Bu sprint", "Şu an yalnızca fissür örtücü ve kanal tedavisi sahneleniyor.");
      return;
    }
    this.kill();
    select.value = String(i);
    const tooth = (clip.teeth && clip.teeth[0]) || "16";
    this.activeTL = fn(this.ctx(), tooth);
    return this.activeTL;
  }

  playPlan() {
    if (!this.playlist.clips.length) return;
    this.kill();
    const run = (i) => {
      if (i >= this.playlist.clips.length) {
        this.caption("Bitti", "İki sinematik tamamlandı.");
        return;
      }
      const tl = this.playClip(i);
      if (!tl) {
        run(i + 1);
        return;
      }
      tl.eventCallback("onComplete", () => run(i + 1));
    };
    run(0);
  }
}

async function loadPlaylist() {
  const q = new URLSearchParams(location.search);
  const items = q.get("items") || "fissure_sealant:16,root_canal:16";
  const demo = new URLSearchParams({
    code: q.get("code") || "",
    teeth: q.get("teeth") || "",
    locale: q.get("locale") || "tr",
    items,
  });
  const res = await fetch(`/v1/viz/demo?${demo}`);
  if (!res.ok) throw new Error("demo playlist failed");
  const pl = await res.json();
  pl.clips = (pl.clips || []).filter((c) => c.code === "fissure_sealant" || c.code === "root_canal");
  if (!pl.clips.length) {
    pl.clips = [
      { code: "fissure_sealant", teeth: ["16"], title: "Fissür Örtücü", moves: [] },
      { code: "root_canal", teeth: ["16"], title: "Kanal tedavisi", moves: [] },
    ];
  }
  return pl;
}

const director = new Director();
const select = document.getElementById("clip");
window.playTreatment = (code, tooth) => {
  const i = director.playlist.clips.findIndex((c) => c.code === code);
  if (i >= 0) {
    if (tooth) director.playlist.clips[i].teeth = [String(tooth)];
    return director.playClip(i);
  }
  const fn = Cinematics[code];
  if (fn) return fn(director.ctx(), tooth || "16");
};

Promise.all([director.loadModel(), loadPlaylist()])
  .then(([, pl]) => {
    director.playlist = pl;
    select.innerHTML = "";
    pl.clips.forEach((c, i) => {
      const o = document.createElement("option");
      o.value = String(i);
      o.textContent = `${c.title} (${(c.teeth || ["16"]).join(",")})`;
      select.appendChild(o);
    });
    director.caption("Hazır", "Fissür örtücü · Kanal tedavisi. Planı oynat.");
    if (new URLSearchParams(location.search).get("autoplay") === "1") director.playPlan();
  })
  .catch((err) => director.caption("Yüklenemedi", err.message));

select.addEventListener("change", () => director.playClip(Number(select.value)));
document.getElementById("play").addEventListener("click", () => director.playPlan());
document.getElementById("prev").addEventListener("click", () => director.playClip(Math.max(0, director.clipIndex - 1)));
document.getElementById("next").addEventListener("click", () => director.playClip(Math.min(director.playlist.clips.length - 1, director.clipIndex + 1)));
