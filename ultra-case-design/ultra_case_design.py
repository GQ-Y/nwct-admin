"""
Luckfox Pico Ultra 开发板 + 4寸屏幕 + 天线 智能家居外壳设计
所有参数都可以在脚本顶部轻松调整
"""

import bpy
import math
import bmesh
from mathutils import Vector

# ==================== 清空场景 ====================
if bpy.context.active_object and bpy.context.active_object.mode != 'OBJECT':
    bpy.ops.object.mode_set(mode='OBJECT')
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete()

# ==================== 单位设置 ====================
bpy.context.scene.unit_settings.system = 'METRIC'
bpy.context.scene.unit_settings.scale_length = 0.001  # 毫米

# ==================== 基础参数 ====================
WALL = 2.0  # 外壳壁厚（mm）

# 开发板尺寸（根据官方文档：50mm x 50mm）
PCB_SIZE = 50.0  # 开发板边长（mm）
PCB_HEIGHT = 2.0  # 开发板厚度（mm，包含元件）

# 外壳尺寸 (Box)
BOX_HEIGHT = 25.0  # 外壳高度（mm，需覆盖开发板和天线）
BOX_X = PCB_SIZE + WALL * 2  # 外壳X方向尺寸
BOX_Y = PCB_SIZE + WALL * 2  # 外壳Y方向尺寸

# ==================== 组装参数 ====================
MOUNT_POST_RADIUS = 4.0   # 组装立柱半径 (mm)
MOUNT_SCREW_DIAMETER = 3.2 # M3 螺丝通孔直径
MOUNT_HOLE_DIAMETER = 2.5  # M3 自攻螺丝底孔直径
MOUNT_HEAD_DIAMETER = 6.0  # 螺丝头直径
MOUNT_HEAD_DEPTH = 3.0     # 螺丝头沉孔深度
ALIGNMENT_RECESS_DEPTH = 1.0 # 对齐凹槽深度

# ==================== 接口开孔参数 ====================
USB_A_WIDTH = 5.12
USB_A_HEIGHT = 14.0
USB_A_DEPTH = 8.0
USB_A_X = -2.0
USB_A_Y = -BOX_Y / 2
USB_A_Z = 12.0

TYPE_C_WIDTH = 8.34
TYPE_C_HEIGHT = 2.56
TYPE_C_DEPTH = 8.0
TYPE_C_X = 0.0
TYPE_C_Y = BOX_Y / 2
TYPE_C_Z = 6.0

RJ45_WIDTH = 16.0
RJ45_HEIGHT = 14.0
RJ45_DEPTH = 18.0
RJ45_X = -12.0
RJ45_Y = -BOX_Y / 2
RJ45_Z = 12.0

ANTENNA_DIAMETER = 8.0
ANTENNA_DEPTH = 15.0
ANTENNA_X = 15.0
ANTENNA_Y = -BOX_Y / 2
ANTENNA_Z = 12.0

# ==================== 屏幕参数 ====================
SCREEN_OUTER = 84.0
SCREEN_INNER = 72.0
SCREEN_HEIGHT = 7.0
SCREEN_LIP = 2.5 # 凹槽深度

# FPC 排线板区域
SCREEN_FPC_HEIGHT = 16.0 
SCREEN_FPC_WIDTH = 32.0
SCREEN_FPC_POCKET_DEPTH = 5.0
SCREEN_CHIN = SCREEN_FPC_HEIGHT + 2.0

# 屏幕与开发板外壳的连接方式
SCREEN_MOUNT_OFFSET = 2.0
SCREEN_SCREW_HOLE_DIAMETER = 1.6
SCREEN_SCREW_OFFSET = 6.0

# ==================== 辅助函数 ====================

def create_cube(x, y, z, lx, ly, lz):
    bpy.ops.mesh.primitive_cube_add(location=(x, y, z))
    obj = bpy.context.object
    obj.scale = (lx / 2, ly / 2, lz / 2)
    bpy.ops.object.transform_apply(scale=True)
    return obj

def create_cylinder(x, y, z, r, h, rot=None):
    bpy.ops.mesh.primitive_cylinder_add(radius=r, depth=h, location=(x, y, z))
    obj = bpy.context.object
    if rot:
        obj.rotation_euler = rot
        bpy.ops.object.transform_apply(rotation=True)
    return obj

def boolean_difference(target, cutter):
    if not target or not cutter: return
    mod = target.modifiers.new(name="Boolean", type='BOOLEAN')
    mod.object = cutter
    mod.operation = 'DIFFERENCE'
    mod.solver = 'EXACT'
    bpy.context.view_layer.objects.active = target
    bpy.ops.object.modifier_apply(modifier=mod.name)
    bpy.data.objects.remove(cutter, do_unlink=True)

def boolean_union(target, joiner):
    if not target or not joiner: return
    mod = target.modifiers.new(name="Boolean", type='BOOLEAN')
    mod.object = joiner
    mod.operation = 'UNION'
    mod.solver = 'EXACT'
    bpy.context.view_layer.objects.active = target
    bpy.ops.object.modifier_apply(modifier=mod.name)
    bpy.data.objects.remove(joiner, do_unlink=True)

# ==================== 创建 Box ====================
box = create_cube(0, 0, BOX_HEIGHT / 2, BOX_X, BOX_Y, BOX_HEIGHT)

# 立柱
POST_POS_X = BOX_X / 2
POST_POS_Y = BOX_Y / 2
post_positions = [
    (POST_POS_X, POST_POS_Y), (-POST_POS_X, POST_POS_Y),
    (-POST_POS_X, -POST_POS_Y), (POST_POS_X, -POST_POS_Y)
]
for px, py in post_positions:
    post = create_cylinder(px, py, BOX_HEIGHT / 2, MOUNT_POST_RADIUS, BOX_HEIGHT)
    boolean_union(box, post)

# 内部挖空
inner = create_cube(0, 0, BOX_HEIGHT / 2 + 1, PCB_SIZE, PCB_SIZE, BOX_HEIGHT - 2)
boolean_difference(box, inner)

# 散热孔
for i in range(5):
    offset = (i - 2) * 6.0
    vent = create_cube(0, offset, 0, 30.0, 3.0, 4.0)
    boolean_difference(box, vent)

# 螺丝孔 (Box)
for px, py in post_positions:
    hole = create_cylinder(px, py, BOX_HEIGHT / 2, MOUNT_SCREW_DIAMETER / 2, BOX_HEIGHT + 2)
    boolean_difference(box, hole)
    head = create_cylinder(px, py, MOUNT_HEAD_DEPTH / 2, MOUNT_HEAD_DIAMETER / 2, MOUNT_HEAD_DEPTH)
    boolean_difference(box, head)

# 接口开孔
usb_a_cut = create_cube(USB_A_X, USB_A_Y + USB_A_DEPTH / 2, USB_A_Z, USB_A_WIDTH, USB_A_DEPTH, USB_A_HEIGHT)
boolean_difference(box, usb_a_cut)

type_c_cut = create_cube(TYPE_C_X, TYPE_C_Y - TYPE_C_DEPTH / 2, TYPE_C_Z, TYPE_C_WIDTH, TYPE_C_DEPTH, TYPE_C_HEIGHT)
boolean_difference(box, type_c_cut)

rj45_cut = create_cube(RJ45_X, RJ45_Y + RJ45_DEPTH / 2, RJ45_Z, RJ45_WIDTH, RJ45_DEPTH, RJ45_HEIGHT)
boolean_difference(box, rj45_cut)

antenna_cut = create_cylinder(ANTENNA_X, ANTENNA_Y + ANTENNA_DEPTH / 2, ANTENNA_Z, ANTENNA_DIAMETER / 2, ANTENNA_DEPTH, (math.pi / 2, 0, 0))
boolean_difference(box, antenna_cut)

# ==================== 创建 Screen Box ====================
screen_box_total_height = SCREEN_OUTER + WALL * 2 + SCREEN_CHIN
screen_box_center_y = -SCREEN_CHIN / 2

screen_box = create_cube(
    0, 
    screen_box_center_y, 
    BOX_HEIGHT + SCREEN_HEIGHT / 2, 
    SCREEN_OUTER + WALL * 2, 
    screen_box_total_height, 
    SCREEN_HEIGHT
)

# 屏幕区域
screen_cut = create_cube(
    0, 
    0, 
    BOX_HEIGHT + SCREEN_HEIGHT / 2 + SCREEN_LIP, 
    SCREEN_INNER, 
    SCREEN_INNER, 
    SCREEN_HEIGHT
)
boolean_difference(screen_box, screen_cut)

# 屏幕嵌入槽 (LIP)
screen_lip = create_cube(
    0, 
    0, 
    (BOX_HEIGHT + SCREEN_HEIGHT) - (SCREEN_LIP + 0.1) / 2,
    SCREEN_OUTER - 4, 
    SCREEN_OUTER - 4, 
    SCREEN_LIP + 0.1
)
boolean_difference(screen_box, screen_lip)

# FPC Pocket
fpc_pocket_y = -SCREEN_OUTER / 2 - SCREEN_FPC_HEIGHT / 2
fpc_pocket_z = (BOX_HEIGHT + SCREEN_HEIGHT) - SCREEN_FPC_POCKET_DEPTH / 2
fpc_pocket = create_cube(
    0,
    fpc_pocket_y,
    fpc_pocket_z,
    SCREEN_FPC_WIDTH,
    SCREEN_FPC_HEIGHT,
    SCREEN_FPC_POCKET_DEPTH
)
boolean_difference(screen_box, fpc_pocket)

# 组装结构 (Align)
align_body = create_cube(0, 0, BOX_HEIGHT + ALIGNMENT_RECESS_DEPTH / 2, BOX_X + 0.4, BOX_Y + 0.4, ALIGNMENT_RECESS_DEPTH)
for px, py in post_positions:
    p = create_cylinder(px, py, BOX_HEIGHT + ALIGNMENT_RECESS_DEPTH / 2, MOUNT_POST_RADIUS + 0.2, ALIGNMENT_RECESS_DEPTH)
    boolean_union(align_body, p)
boolean_difference(screen_box, align_body)

# 组装结构 (Screw Holes)
for px, py in post_positions:
    screw_hole_screen = create_cylinder(
        px, 
        py, 
        BOX_HEIGHT + SCREEN_HEIGHT / 2, 
        MOUNT_HOLE_DIAMETER / 2, 
        SCREEN_HEIGHT - 2 
    )
    boolean_difference(screen_box, screw_hole_screen)

# ==================== FPC排线通道 (直通 + 底部开口) ====================
# 去除横梁 (Connect Lip and Pocket)
# 位于 Screen Lip 底部边缘 (-40) 和 FPC Pocket 顶部边缘 (-42) 之间
# 创建一个连接体打通两者

# 连接体参数
connector_width = SCREEN_FPC_WIDTH # 32.0
connector_y = -SCREEN_OUTER / 2 # -42 (Approx border)
connector_len = 6.0 # Enough to bridge the gap
connector_z = fpc_pocket_z # Same depth as pocket

fpc_connector = create_cube(
    0,
    connector_y,
    connector_z,
    connector_width,
    connector_len,
    SCREEN_FPC_POCKET_DEPTH
)
boolean_difference(screen_box, fpc_connector)

# 垂直开口 (仅在 Box 内部区域打开底部)
# 连接 Pocket (now extended) 和 Box 内部
drop_y_start = -PCB_SIZE / 2 # -25 (Box Wall)
drop_y_end = -SCREEN_OUTER / 2 - 5 # Inside Pocket area
drop_len = abs(drop_y_end - drop_y_start)
drop_center_y = (drop_y_start + drop_y_end) / 2

# 底部保留厚度
floor_thickness = 1.0
drop_z_pos = BOX_HEIGHT + floor_thickness / 2

wire_drop = create_cube(
    0,
    drop_center_y,
    drop_z_pos,
    SCREEN_FPC_WIDTH - 4, # Slightly narrower drop hole
    drop_len,
    floor_thickness + 0.1
)
boolean_difference(screen_box, wire_drop)

# 屏幕固定螺丝孔
for x, y in [
    (SCREEN_OUTER/2 - 6, SCREEN_OUTER/2 - 6),
    (-SCREEN_OUTER/2 + 6, SCREEN_OUTER/2 - 6),
    (-SCREEN_OUTER/2 + 6, -SCREEN_OUTER/2 + 6),
    (SCREEN_OUTER/2 - 6, -SCREEN_OUTER/2 + 6)
]:
    screw_hole = create_cylinder(x, y, BOX_HEIGHT + SCREEN_HEIGHT/2, 0.8, SCREEN_HEIGHT)
    boolean_difference(screen_box, screw_hole)

# 设置父子关系
if screen_box and box:
    try:
        screen_box.parent = box
    except:
        pass

# 倒角
if box:
    bev1 = box.modifiers.new("bevel", 'BEVEL')
    bev1.width = 2.0
    bev1.segments = 4
    bev1.limit_method = 'ANGLE'
    bev1.angle_limit = math.radians(35)
    bpy.context.view_layer.objects.active = box
    bpy.ops.object.modifier_apply(modifier=bev1.name)

if screen_box:
    bev2 = screen_box.modifiers.new("bevel", 'BEVEL')
    bev2.width = 0.6
    bev2.segments = 2
    bev2.limit_method = 'ANGLE'
    bpy.context.view_layer.objects.active = screen_box
    bpy.ops.object.modifier_apply(modifier=bev2.name)

# 平滑
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.shade_smooth()

print("=" * 60)
print("外壳设计完成！(去横梁直通版)")
print("=" * 60)
print(f"已打通屏幕凹槽与FPC口袋之间的横梁")
print(f"保留了进入Box内部的垂直落孔")
print("=" * 60)