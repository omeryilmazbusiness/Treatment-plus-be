import * as THREE from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import { GLTFLoader } from "three/addons/loaders/GLTFLoader.js";
import { EffectComposer } from "three/addons/postprocessing/EffectComposer.js";
import { RenderPass } from "three/addons/postprocessing/RenderPass.js";
import { BokehPass } from "three/addons/postprocessing/BokehPass.js";
import { OutputPass } from "three/addons/postprocessing/OutputPass.js";
import gsap from "gsap";
import { Cinematics, playFissureSealant, playRootCanal, ensureOpenMouth } from "./cinematics.js?v=20";

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
    this.scene.background = new THREE.Color(0x070d18);
    this.camera = new THREE.PerspectiveCamera(32, 1, 0.005, 80);
    this.renderer = new THREE.WebGLRenderer({ antialias: true, powerPreference: "high-performance" });
    this.renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
    this.renderer.outputColorSpace = THREE.SRGBColorSpace;
    this.renderer.toneMapping = THREE.ACESFilmicToneMapping;
    this.renderer.toneMappingExposure = 1.08;
    this.renderer.localClippingEnabled = false;
    document.getElementById("stage").appendChild(this.renderer.domElement);
    this.controls = new OrbitControls(this.camera, this.renderer.domElement);
    this.controls.enableDamping = true;
    this.scene.add(new THREE.HemisphereLight(0xdbeafe, 0x2a1a16, 0.95));
    const key = new THREE.DirectionalLight(0xfff7ed, 1.25);
    key.position.set(2.2, 3.4, 2.8);
    this.scene.add(key, new THREE.AmbientLight(0xffffff, 0.22));
    this.composer = new EffectComposer(this.renderer);
    this.composer.addPass(new RenderPass(this.scene, this.camera));
    try {
      this.bokeh = new BokehPass(this.scene, this.camera, { focus: 1.4, aperture: 0.00018, maxblur: 0.004 });
      this.composer.addPass(this.bokeh);
    } catch (err) {
      console.warn("[viz] DoF unavailable", err);
      this.bokeh = null;
    }
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
    snapshotMats(this.root);
    this.scene.add(this.root);
    inspectModel(this.root);
    ensureOpenMouth(this.root, gsap, 0.16, 0);
    window.__director = this;
    const hinge = this.root.getObjectByName("jaw_hinge_pivot");
    const jaw = this.root.getObjectByName("jawLower_midres");
    console.info("[viz] mouth", {
      hinge: hinge?.name,
      hingeRx: hinge?.rotation.x,
      jawParent: jaw?.parent?.name,
    });
    const focus = this.root.getObjectByName("TeethTongueSet_midres")
      || this.root.getObjectByName("TeethUpper_midres")
      || this.root;
    const box = new THREE.Box3().setFromObject(focus);
    const c = box.getCenter(new THREE.Vector3());
    const size = box.getSize(new THREE.Vector3());
    const dist = Math.max(size.length() * 2.6, 0.26);
    this.camera.position.set(c.x + dist * 0.12, c.y + dist * 0.22, c.z + dist * 0.98);
    this.controls.target.copy(c);
    this.camera.lookAt(c);
    this.caption("Hazır", "Fissür örtücü ve kanal tedavisi sinematikleri hazır.");
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
