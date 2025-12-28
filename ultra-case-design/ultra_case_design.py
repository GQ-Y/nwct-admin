"""
Luckfox Pico Ultra 开发板 + 4寸屏幕 + 天线 智能家居外壳设计
所有参数都可以在脚本顶部轻松调整
"""

import bpy
import math
import bmesh
from mathutils import Vector

# ==================== 清空场景 ====================
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

# 外壳尺寸
BOX_HEIGHT = 25.0  # 外壳高度（mm，需覆盖开发板和天线）
BOX_X = PCB_SIZE + WALL * 2  # 外壳X方向尺寸
BOX_Y = PCB_SIZE + WALL * 2  # 外壳Y方向尺寸

# ==================== 接口开孔参数（可编辑） ====================

# USB Type-A 接口（底部，竖着，在RJ45孔旁边）
USB_A_WIDTH = 5.12  # USB-A 接口宽度（mm，标准尺寸，竖着时是宽度）
USB_A_HEIGHT = 14.0  # USB-A 接口高度（mm，改为14.0与RJ45一致，方便打印）
USB_A_DEPTH = 8.0  # 开孔深度（mm）
USB_A_X = -2.0  # X位置（相对于开发板中心，在RJ45孔右侧，向左移动更靠近网口，mm）
USB_A_Y = -BOX_Y / 2  # Y位置（底部边缘，mm，通常不需要修改）
USB_A_Z = 12.0  # Z位置（高度，mm，根据接口在开发板上的实际高度调整）

# USB Type-C 接口（顶部中心，根据实际开发板位置调整）
TYPE_C_WIDTH = 8.34  # Type-C 接口宽度（mm，标准尺寸）
TYPE_C_HEIGHT = 2.56  # Type-C 接口高度（mm，标准尺寸）
TYPE_C_DEPTH = 8.0  # 开孔深度（mm）
TYPE_C_RADIUS = 0.4  # Type-C 圆角半径（mm，圆角矩形的圆角半径）
TYPE_C_X = 0.0  # X位置（相对于开发板中心，正数=右，负数=左，mm）
TYPE_C_Y = BOX_Y / 2  # Y位置（顶部边缘，mm，通常不需要修改）
TYPE_C_Z = 6.0  # Z位置（高度，mm，根据接口在开发板上的实际高度调整）

# RJ45 网口（底部左侧，根据实际开发板位置调整）
RJ45_WIDTH = 16.0  # 网口宽度（mm，标准尺寸，可调整为14.5-16.0）
RJ45_HEIGHT = 14.0  # 网口高度（mm，标准尺寸，可调整为14.0-16.0）
RJ45_DEPTH = 18.0  # 开孔深度（mm）
RJ45_X = -12.0  # X位置（相对于开发板中心，正数=右，负数=左，mm）
RJ45_Y = -BOX_Y / 2  # Y位置（底部边缘，mm，通常不需要修改）
RJ45_Z = 12.0  # Z位置（高度，mm，根据接口在开发板上的实际高度调整）

# 天线开孔（底部侧壁，靠右位置）
ANTENNA_DIAMETER = 8.0  # 天线开孔直径（mm，标准U.FL/IPEX连接器，可调整为6-10mm）
ANTENNA_DEPTH = 15.0  # 开孔深度（mm）
ANTENNA_X = 15.0  # X位置（相对于开发板中心，靠右，mm）
ANTENNA_Y = -BOX_Y / 2  # Y位置（底部边缘，mm）
ANTENNA_Z = 12.0  # Z位置（高度，mm，与RJ45/USB-A高度一致）

# ==================== 屏幕参数 ====================
SCREEN_OUTER = 84.0  # 屏幕外部尺寸（mm）
SCREEN_INNER = 72.0  # 屏幕可视区域尺寸（mm）
SCREEN_HEIGHT = 7.0  # 屏幕厚度（mm）
SCREEN_LIP = 1.1  # 屏幕边缘高度（mm）

# 屏幕与开发板外壳的连接方式
SCREEN_MOUNT_OFFSET = 2.0  # 屏幕安装偏移（mm，用于卡扣或螺丝固定）
SCREEN_SCREW_HOLE_DIAMETER = 1.6  # 屏幕固定螺丝孔直径（mm）
SCREEN_SCREW_OFFSET = 6.0  # 螺丝孔距离屏幕边缘的距离（mm）

# ==================== 辅助函数 ====================

def create_cube(x, y, z, lx, ly, lz):
    """创建立方体"""
    bpy.ops.mesh.primitive_cube_add(location=(x, y, z))
    obj = bpy.context.object
    obj.scale = (lx / 2, ly / 2, lz / 2)
    bpy.ops.object.transform_apply(scale=True)
    return obj

def create_cylinder(x, y, z, r, h, rot=None):
    """创建圆柱体"""
    bpy.ops.mesh.primitive_cylinder_add(radius=r, depth=h, location=(x, y, z))
    obj = bpy.context.object
    if rot:
        obj.rotation_euler = rot
        bpy.ops.object.transform_apply(rotation=True)
    return obj

def create_rounded_rectangle(x, y, z, width, height, depth, radius):
    """创建真正的圆角矩形（使用bmesh构建，圆润的矩形形状）"""
    mesh = bpy.data.meshes.new(name="RoundedRect")
    bm = bmesh.new()
    
    # 计算圆角矩形的尺寸
    w = width / 2
    h = height / 2
    r = min(radius, w, h)  # 确保圆角半径不超过矩形的一半
    
    # 创建圆角矩形的顶点（在XY平面上）
    # 使用8个点来近似每个圆角（更平滑）
    segments = 8
    
    verts = []
    
    # 四个圆角的中心点
    corners = [
        (w - r, h - r),   # 右上
        (-w + r, h - r),  # 左上
        (-w + r, -h + r), # 左下
        (w - r, -h + r)   # 右下
    ]
    
    # 创建圆角矩形轮廓
    for i in range(4):
        cx, cy = corners[i]
        start_angle = i * math.pi / 2 + math.pi / 2
        
        # 添加圆角点
        for j in range(segments + 1):
            angle = start_angle + (math.pi / 2) * (j / segments)
            px = cx + r * math.cos(angle)
            py = cy + r * math.sin(angle)
            verts.append((px, py, 0))
    
    # 创建面
    for v in verts:
        bm.verts.new(v)
    
    bm.verts.ensure_lookup_table()
    
    # 创建面（使用索引）
    face_verts = [bm.verts[i] for i in range(len(verts))]
    bm.faces.new(face_verts)
    
    # 更新mesh
    bm.to_mesh(mesh)
    bm.free()
    
    # 创建对象
    obj = bpy.data.objects.new("RoundedRect", mesh)
    bpy.context.collection.objects.link(obj)
    
    # 设置位置
    obj.location = (x, y, z)
    
    # 挤出深度
    bpy.context.view_layer.objects.active = obj
    bpy.ops.object.mode_set(mode='EDIT')
    bpy.ops.mesh.extrude_region_move(TRANSFORM_OT_translate={"value": (0, 0, depth)})
    bpy.ops.object.mode_set(mode='OBJECT')
    
    # 移动到正确位置（考虑深度）
    obj.location.z = z + depth / 2
    
    return obj

def boolean_difference(target, cutter):
    """布尔差集运算"""
    mod = target.modifiers.new(name="Boolean", type='BOOLEAN')
    mod.object = cutter
    mod.operation = 'DIFFERENCE'
    mod.solver = 'EXACT'
    bpy.context.view_layer.objects.active = target
    bpy.ops.object.modifier_apply(modifier=mod.name)
    bpy.data.objects.remove(cutter, do_unlink=True)

# ==================== 创建开发板外壳主体 ====================

# 外壳主体
box = create_cube(0, 0, BOX_HEIGHT / 2, BOX_X, BOX_Y, BOX_HEIGHT)

# 内部空间（减去内部空间）
inner = create_cube(0, 0, BOX_HEIGHT / 2 + 1, PCB_SIZE, PCB_SIZE, BOX_HEIGHT - 2)
boolean_difference(box, inner)

# ==================== 创建接口开孔 ====================

# USB Type-A 开孔（底部，从底部边缘向内延伸）
# 开孔中心位置：从底部边缘向内偏移 DEPTH/2
usb_a_cut = create_cube(
    USB_A_X, 
    USB_A_Y + USB_A_DEPTH / 2,  # 从底部边缘向内延伸
    USB_A_Z, 
    USB_A_WIDTH, 
    USB_A_DEPTH, 
    USB_A_HEIGHT
)
boolean_difference(box, usb_a_cut)

# USB Type-C 开孔（顶部，从顶部边缘向内延伸，真正的圆角矩形形状）
# 开孔中心位置：从顶部边缘向内偏移 DEPTH/2
type_c_cut = create_rounded_rectangle(
    TYPE_C_X, 
    TYPE_C_Y - TYPE_C_DEPTH / 2,  # 从顶部边缘向内延伸
    TYPE_C_Z, 
    TYPE_C_WIDTH, 
    TYPE_C_HEIGHT, 
    TYPE_C_DEPTH, 
    TYPE_C_RADIUS  # 圆角半径
)
boolean_difference(box, type_c_cut)

# RJ45 网口开孔（底部，从底部边缘向内延伸）
# 开孔中心位置：从底部边缘向内偏移 DEPTH/2
rj45_cut = create_cube(
    RJ45_X, 
    RJ45_Y + RJ45_DEPTH / 2,  # 从底部边缘向内延伸
    RJ45_Z, 
    RJ45_WIDTH, 
    RJ45_DEPTH, 
    RJ45_HEIGHT
)
boolean_difference(box, rj45_cut)

# 天线开孔（底部侧壁，从底部边缘向内延伸）
# 开孔中心位置：从底部边缘向内偏移 DEPTH/2
antenna_cut = create_cylinder(
    ANTENNA_X, 
    ANTENNA_Y + ANTENNA_DEPTH / 2,  # 从底部边缘向内延伸
    ANTENNA_Z, 
    ANTENNA_DIAMETER / 2, 
    ANTENNA_DEPTH, 
    (math.pi / 2, 0, 0)  # 绕X轴旋转90度，使圆柱体沿Y轴方向
)
boolean_difference(box, antenna_cut)

# ==================== 创建屏幕外壳 ====================

# 屏幕外壳主体
screen_box = create_cube(
    0, 
    0, 
    BOX_HEIGHT + SCREEN_HEIGHT / 2, 
    SCREEN_OUTER + WALL * 2, 
    SCREEN_OUTER + WALL * 2, 
    SCREEN_HEIGHT
)

# 屏幕显示区域开孔
screen_cut = create_cube(
    0, 
    0, 
    BOX_HEIGHT + SCREEN_HEIGHT / 2 + SCREEN_LIP, 
    SCREEN_INNER, 
    SCREEN_INNER, 
    SCREEN_HEIGHT
)
boolean_difference(screen_box, screen_cut)

# 屏幕边缘（LIP）
screen_lip = create_cube(
    0, 
    0, 
    BOX_HEIGHT + SCREEN_HEIGHT - SCREEN_LIP / 2, 
    SCREEN_OUTER - 4, 
    SCREEN_OUTER - 4, 
    SCREEN_LIP
)
boolean_difference(screen_box, screen_lip)

# FPC排线开孔（屏幕底部）
fpc_cut = create_cube(
    0, 
    -SCREEN_OUTER / 2 + 10, 
    BOX_HEIGHT + SCREEN_HEIGHT / 2, 
    16, 
    8, 
    SCREEN_HEIGHT + 2
)
boolean_difference(screen_box, fpc_cut)

# 屏幕固定螺丝孔（四个角）
for x, y in [
    (SCREEN_OUTER / 2 - SCREEN_SCREW_OFFSET, SCREEN_OUTER / 2 - SCREEN_SCREW_OFFSET),
    (-SCREEN_OUTER / 2 + SCREEN_SCREW_OFFSET, SCREEN_OUTER / 2 - SCREEN_SCREW_OFFSET),
    (-SCREEN_OUTER / 2 + SCREEN_SCREW_OFFSET, -SCREEN_OUTER / 2 + SCREEN_SCREW_OFFSET),
    (SCREEN_OUTER / 2 - SCREEN_SCREW_OFFSET, -SCREEN_OUTER / 2 + SCREEN_SCREW_OFFSET)
]:
    screw_hole = create_cylinder(
        x, 
        y, 
        BOX_HEIGHT + SCREEN_HEIGHT / 2, 
        SCREEN_SCREW_HOLE_DIAMETER / 2, 
        SCREEN_HEIGHT
    )
    boolean_difference(screen_box, screw_hole)

# 将屏幕外壳与开发板外壳关联（但不合并，方便单独编辑）
screen_box.parent = box

# ==================== 添加倒角（美化） ====================

# 开发板外壳倒角
bev1 = box.modifiers.new("bevel", 'BEVEL')
bev1.width = 0.6
bev1.segments = 2
bev1.profile = 0.7
bev1.limit_method = 'ANGLE'
bpy.context.view_layer.objects.active = box
bpy.ops.object.modifier_apply(modifier=bev1.name)

# 屏幕外壳倒角
bev2 = screen_box.modifiers.new("bevel", 'BEVEL')
bev2.width = 0.6
bev2.segments = 2
bev2.profile = 0.7
bev2.limit_method = 'ANGLE'
bpy.context.view_layer.objects.active = screen_box
bpy.ops.object.modifier_apply(modifier=bev2.name)

# ==================== 平滑处理 ====================
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.shade_smooth()

print("=" * 60)
print("外壳设计完成！")
print("=" * 60)
print(f"开发板外壳尺寸: {BOX_X} x {BOX_Y} x {BOX_HEIGHT} mm")
print(f"屏幕外壳尺寸: {SCREEN_OUTER + WALL * 2} x {SCREEN_OUTER + WALL * 2} x {SCREEN_HEIGHT} mm")
print("=" * 60)
print("所有参数都可以在脚本顶部轻松调整")
print("=" * 60)

