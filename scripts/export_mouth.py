# Export the existing Lashko mouth as GLB. Does not create geometry.
# blender --background --python scripts/export_mouth.py -- /path/to.blend /path/to/mouth.glb
import json
import sys

import bpy

argv = sys.argv[sys.argv.index("--") + 1 :] if "--" in sys.argv else []
src = argv[0] if len(argv) > 0 else "/Users/omeryilmaz/Desktop/Teeth_Blender_Packed.blend"
dst = argv[1] if len(argv) > 1 else "internal/viz/player/static/mouth.glb"

bpy.ops.wm.open_mainfile(filepath=src)

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
    hide = "lowres" in n or obj.type not in {"MESH", "EMPTY", "ARMATURE"}
    obj.hide_set(hide)
    obj.hide_render = hide
    obj.hide_viewport = hide

bpy.ops.export_scene.gltf(
    filepath=dst,
    export_format="GLB",
    use_visible=True,
    export_apply=True,
    export_texcoords=True,
    export_normals=True,
    export_materials="EXPORT",
    export_cameras=False,
    export_lights=False,
)
print("WROTE", dst)
