# Faithful export of Teeth_Blender_Packed.blend → mouth.glb.
# Does not author geometry. Converts only shaders glTF cannot store (Glass BSDF),
# then writes lights, cameras, tangents, packed maps, and volume/transmission.
# blender --background --python scripts/export_mouth.py -- /path/to.blend /path/to/mouth.glb
import json
import sys

import bpy

try:
    from io_scene_gltf2.blender.com.material_helpers import create_settings_group, get_gltf_node_name
except ImportError:
    def get_gltf_node_name():
        return "glTF Material Output"

    def create_settings_group(name):
        group = bpy.data.node_groups.new(name, "ShaderNodeTree")
        group.interface.new_socket("Occlusion", socket_type="NodeSocketFloat")
        thick = group.interface.new_socket("Thickness", socket_type="NodeSocketFloat")
        thick.default_value = 0.0
        group.nodes.new("NodeGroupOutput")
        group.nodes.new("NodeGroupInput").location = (-200, 0)
        return group

argv = sys.argv[sys.argv.index("--") + 1 :] if "--" in sys.argv else []
src = argv[0] if len(argv) > 0 else "/Users/omeryilmaz/Desktop/General/Teeth_Blender_Packed.blend"
dst = argv[1] if len(argv) > 1 else "internal/viz/player/static/mouth.glb"

bpy.ops.wm.open_mainfile(filepath=src)


def sock(node, name):
    s = node.inputs.get(name)
    if s is None:
        return None
    try:
        v = s.default_value
        return list(v)
    except TypeError:
        return s.default_value


def ensure_gltf_thickness(mat, thickness):
    if not mat or not mat.use_nodes or thickness <= 0:
        return
    name = get_gltf_node_name()
    group = bpy.data.node_groups.get(name) or create_settings_group(name)
    nt = mat.node_tree
    node = next((n for n in nt.nodes if n.type == "GROUP" and n.node_tree == group), None)
    if node is None:
        node = nt.nodes.new("ShaderNodeGroup")
        node.node_tree = group
        node.location = (400, -200)
    if "Thickness" in node.inputs:
        node.inputs["Thickness"].default_value = thickness


def bypass_unused_color_mix(mat):
    if not mat or not mat.use_nodes:
        return
    nt = mat.node_tree
    princ = next((n for n in nt.nodes if n.type == "BSDF_PRINCIPLED"), None)
    mix = next((n for n in nt.nodes if n.type == "MIX"), None)
    if not princ or not mix:
        return
    fac = mix.inputs[0].default_value if mix.inputs else 0
    a_link = next((l for l in nt.links if l.to_node == mix and l.to_socket == mix.inputs[6]), None)
    if a_link is None:
        a_link = next((l for l in nt.links if l.to_node == mix and l.to_socket.name in {"A", "Color1"}), None)
    if fac == 0 and a_link is not None:
        for link in list(nt.links):
            if link.from_node == mix and link.to_node == princ and link.to_socket.name == "Base Color":
                nt.links.remove(link)
        nt.links.new(a_link.from_socket, princ.inputs["Base Color"])


def convert_saliva():
    mat = bpy.data.materials.get("saliva")
    if not mat or not mat.use_nodes:
        return
    nt = mat.node_tree
    out = next((n for n in nt.nodes if n.type == "OUTPUT_MATERIAL"), None)
    mask = next((n for n in nt.nodes if n.type == "TEX_IMAGE" and n.image and "MASK" in n.image.name), None)
    normal_tex = next((n for n in nt.nodes if n.type == "TEX_IMAGE" and n.image and "NORMAL" in n.image.name), None)
    glass = next((n for n in nt.nodes if n.type == "BSDF_GLASS"), None)
    value = next((n for n in nt.nodes if n.type == "VALUE"), None)
    thickness = value.outputs[0].default_value if value else 0.01
    ior = sock(glass, "IOR") if glass else 1.33
    roughness = sock(glass, "Roughness") if glass else 0.0
    color = sock(glass, "Color") if glass else [1, 1, 1, 1]

    princ = nt.nodes.new("ShaderNodeBsdfPrincipled")
    princ.location = (240, 40)
    princ.inputs["Metallic"].default_value = 0
    princ.inputs["Roughness"].default_value = roughness if roughness is not None else 0.0
    princ.inputs["IOR"].default_value = ior if isinstance(ior, (int, float)) else 1.33
    if color:
        princ.inputs["Base Color"].default_value = color
    if "Transmission Weight" in princ.inputs:
        princ.inputs["Transmission Weight"].default_value = 1
    elif "Transmission" in princ.inputs:
        princ.inputs["Transmission"].default_value = 1
    if "Alpha" in princ.inputs:
        princ.inputs["Alpha"].default_value = 1

    if out:
        for link in list(nt.links):
            if link.to_node == out and link.to_socket.name == "Surface":
                nt.links.remove(link)
        nt.links.new(princ.outputs["BSDF"], out.inputs["Surface"])
    if mask:
        nt.links.new(mask.outputs["Color"], princ.inputs["Alpha"])
        mat.blend_method = "BLEND"
        if hasattr(mat, "use_backface_culling"):
            mat.use_backface_culling = False
    if normal_tex:
        nrm = next((n for n in nt.nodes if n.type == "NORMAL_MAP"), None) or nt.nodes.new("ShaderNodeNormalMap")
        if not any(l.to_node == nrm for l in nt.links):
            nt.links.new(normal_tex.outputs["Color"], nrm.inputs["Color"])
        nt.links.new(nrm.outputs["Normal"], princ.inputs["Normal"])
    ensure_gltf_thickness(mat, thickness if thickness else 0.01)


def encode_teeth_volume():
    mat = bpy.data.materials.get("teeth_tongue")
    if not mat or not mat.use_nodes:
        return
    bypass_unused_color_mix(mat)
    princ = next((n for n in mat.node_tree.nodes if n.type == "BSDF_PRINCIPLED"), None)
    weight = sock(princ, "Subsurface Weight") if princ else 0
    radius = sock(princ, "Subsurface Radius") if princ else [0.5, 0.2, 0.1]
    if not weight:
        return
    ensure_gltf_thickness(mat, 0.012)
    nt = mat.node_tree
    out = next((n for n in nt.nodes if n.type == "OUTPUT_MATERIAL"), None)
    if out is None:
        return
    absn = nt.nodes.new("ShaderNodeVolumeAbsorption")
    absn.location = (240, -260)
    if radius:
        absn.inputs["Color"].default_value = [radius[0], radius[1], radius[2], 1.0]
    absn.inputs["Density"].default_value = 8.0 * float(weight)
    if "Volume" in out.inputs:
        nt.links.new(absn.outputs["Volume"], out.inputs["Volume"])


report = []
for obj in bpy.data.objects:
    vgs = []
    if obj.type == "MESH":
        vgs = [g.name for g in getattr(obj, "vertex_groups", [])]
    report.append(
        {
            "name": obj.name,
            "type": obj.type,
            "hide": bool(obj.hide_get()),
            "verts": len(obj.data.vertices) if obj.type == "MESH" else 0,
            "vertex_groups": vgs,
        }
    )
print("MOUTH_GRAPH=" + json.dumps(report, ensure_ascii=False))

for obj in bpy.data.objects:
    n = obj.name.lower()
    keep = obj.type in {"MESH", "EMPTY", "ARMATURE", "LIGHT", "CAMERA"}
    hide = "lowres" in n or not keep
    obj.hide_set(hide)
    obj.hide_render = hide
    obj.hide_viewport = hide

convert_saliva()
encode_teeth_volume()

bpy.ops.export_scene.gltf(
    filepath=dst,
    export_format="GLB",
    use_visible=True,
    export_apply=True,
    export_texcoords=True,
    export_normals=True,
    export_tangents=True,
    export_materials="EXPORT",
    export_unused_images=True,
    export_unused_textures=True,
    export_cameras=True,
    export_lights=True,
    export_extras=True,
    export_original_specular=True,
    export_image_format="AUTO",
)
print("WROTE", dst)
