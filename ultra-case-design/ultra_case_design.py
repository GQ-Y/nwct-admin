"""
Luckfox Pico Ultra + 4寸屏幕 - 复古电脑风格外壳 (Retro PC Style)
特点：
1. "大背头" (Big Back) CRT显示器造型
2. 分体式设计：前脸 (Bezel) + 后壳 (Rear Shell)
3. 优化：顶部封闭（无孔），所有线缆从底部导出 (Hidden Cable Routing)
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

# ==================== 核心参数 ====================
WALL = 3.0 # 壁厚

# 屏幕尺寸
SCREEN_W = 84.0 
SCREEN_H = 84.0 
SCREEN_VISIBLE_W = 72.0
SCREEN_VISIBLE_H = 72.0

# 外壳整体造型尺寸
CASE_WIDTH = 100.0
CASE_HEIGHT = 110.0
CASE_DEPTH = 80.0   # 后壳深度 (Y: 0 to 80)

# PCB 位置参数
PCB_SIZE = 50.0
# PCB 安装在垂直居中，靠后壁的位置
# Y = Back Wall Inner Face - Standoff
STANDOFF_H = 6.0
PCB_Y_POS = (CASE_DEPTH - WALL) - STANDOFF_H # 71.0
# Z = 0 (Center)
# Top Edge Z = +25, Bottom Edge Z = -25
# Case Top Inner Z = +52, Bottom Inner Z = -52
# Top Clearance = 27mm (Enough for Type-C loop)

# ==================== 辅助函数 (绝对坐标版) ====================

def create_block(x_min, x_max, y_min, y_max, z_min, z_max, name="Block"):
    """使用绝对边界坐标创建立方体"""
    x = (x_min + x_max) / 2
    y = (y_min + y_max) / 2
    z = (z_min + z_max) / 2
    lx = x_max - x_min
    ly = y_max - y_min
    lz = z_max - z_min
    
    bpy.ops.mesh.primitive_cube_add(location=(x, y, z))
    obj = bpy.context.object
    obj.scale = (lx / 2, ly / 2, lz / 2)
    bpy.ops.object.transform_apply(scale=True)
    obj.name = name
    return obj

def create_cylinder_axis(x, y, z, r, h, axis='Z', name="Cylinder"):
    """创建圆柱"""
    rot = (0,0,0)
    if axis == 'X': rot = (0, math.radians(90), 0)
    if axis == 'Y': rot = (math.radians(90), 0, 0)
    
    bpy.ops.mesh.primitive_cylinder_add(radius=r, depth=h, location=(x, y, z))
    obj = bpy.context.object
    obj.rotation_euler = rot
    bpy.ops.object.transform_apply(rotation=True)
    obj.name = name
    return obj

def boolean_op(target, tool, operation='DIFFERENCE'):
    if not target or not tool: return
    mod = target.modifiers.new(name="Boolean", type='BOOLEAN')
    mod.object = tool
    mod.operation = operation
    mod.solver = 'EXACT'
    bpy.context.view_layer.objects.active = target
    bpy.ops.object.modifier_apply(modifier=mod.name)
    bpy.data.objects.remove(tool, do_unlink=True)

def add_bevel(obj, width=1.0):
    bev = obj.modifiers.new("Bevel", 'BEVEL')
    bev.width = width
    bev.segments = 3
    bev.limit_method = 'ANGLE'
    bpy.context.view_layer.objects.active = obj
    bpy.ops.object.modifier_apply(modifier=bev.name)

# ==================== 1. 创建后壳 (Rear Shell) ====================
# 范围: X[-50, 50], Y[0, 80], Z[-55, 55]

# 实体
rear_shell = create_block(
    -CASE_WIDTH/2, CASE_WIDTH/2,
    0, CASE_DEPTH,
    -CASE_HEIGHT/2, CASE_HEIGHT/2,
    "RearShell_Body"
)

# 锥度切削 (Taper) - CRT 风格
taper_l = create_block(
    -CASE_WIDTH - 20, -CASE_WIDTH/2 + 5, 
    10, CASE_DEPTH + 10, 
    -CASE_HEIGHT, CASE_HEIGHT,
    "TaperL"
)
taper_l.rotation_euler = (0, 0, math.radians(-10))
taper_l.location.x -= 10
boolean_op(rear_shell, taper_l)

taper_r = create_block(
    CASE_WIDTH/2 - 5, CASE_WIDTH + 20,
    10, CASE_DEPTH + 10,
    -CASE_HEIGHT, CASE_HEIGHT,
    "TaperR"
)
taper_r.rotation_euler = (0, 0, math.radians(10))
taper_r.location.x += 10
boolean_op(rear_shell, taper_r)

taper_t = create_block(
    -CASE_WIDTH, CASE_WIDTH,
    20, CASE_DEPTH + 20,
    CASE_HEIGHT/2 - 5, CASE_HEIGHT + 20,
    "TaperT"
)
taper_t.rotation_euler = (math.radians(10), 0, 0)
boolean_op(rear_shell, taper_t)


# 挖空内部 (Hollow)
# 留出 Back Wall (Y=80) 和 Top Wall (Z=55)
# **关键修改**: 顶部不再开孔，保持 Top Wall 完整
inner_cut = create_block(
    -CASE_WIDTH/2 + WALL, CASE_WIDTH/2 - WALL,
    -1.0, CASE_DEPTH - WALL,
    -CASE_HEIGHT/2 + WALL, CASE_HEIGHT/2 - WALL,
    "InnerCut"
)
boolean_op(rear_shell, inner_cut)


# ==================== 2. 开发板安装 (PCB Mounting) ====================
# 安装在后壁内侧
# PCB 底部边缘 Z = -25
# PCB 顶部边缘 Z = +25

boss_spacing = 50.0 / 2 - 2.5 # 22.5
boss_coords = [
    (-boss_spacing, -boss_spacing), 
    (boss_spacing, -boss_spacing),
    (-boss_spacing, boss_spacing), 
    (boss_spacing, boss_spacing)
]

for dx, dz in boss_coords:
    pillar = create_cylinder_axis(
        dx, 
        (CASE_DEPTH - WALL) - STANDOFF_H/2, 
        dz, 
        3.0, 
        STANDOFF_H, 
        'Y',
        "Standoff"
    )
    boolean_op(rear_shell, pillar, 'UNION')
    
    hole = create_cylinder_axis(
        dx,
        (CASE_DEPTH - WALL) - STANDOFF_H/2,
        dz,
        1.2, 
        STANDOFF_H + 2.0,
        'Y',
        "ScrewHole"
    )
    boolean_op(rear_shell, hole)

# ==================== 3. 底部接口与走线 (Bottom IO) ====================
# 所有接口都在 PCB 竖直平面上 (Y=71)
# 顶部 Type-C (Z=+25): 需要内部留空，线缆向下走
# 底部 USB/RJ45 (Z=-25): 直接向下开口

# 底部大开口 (Cable Bay)
# 打通底部，覆盖 USB, RJ45 以及 额外的 Type-C 走线空间
# X range: Covers PCB width (-25 to +25) plus extra space
# Y range: Around PCB plane
# Z range: Through floor

cable_bay = create_block(
    -35, 35, # 70mm wide opening
    PCB_Y_POS - 15, PCB_Y_POS + 15, # 30mm deep opening around PCB
    -CASE_HEIGHT/2 - 5, -CASE_HEIGHT/2 + WALL + 5, # Through floor
    "CableBay"
)
boolean_op(rear_shell, cable_bay)

# 顶部 Type-C **不** 开孔
# 内部空间高度: Case Height 110. Inner Height ~104. Half ~52.
# PCB Top Edge = +25.
# Clearance = 52 - 25 = 27mm.
# 这足够插入直头或弯头 Type-C 线并向下弯曲。


# ==================== 4. 前脸 (Front Bezel) ====================
BEZEL_DEPTH = 10.0
front_bezel = create_block(
    -CASE_WIDTH/2, CASE_WIDTH/2,
    -BEZEL_DEPTH, 0,
    -CASE_HEIGHT/2, CASE_HEIGHT/2,
    "FrontBezel"
)

screen_hole = create_block(
    -SCREEN_VISIBLE_W/2, SCREEN_VISIBLE_W/2,
    -BEZEL_DEPTH - 1, 1,
    -SCREEN_VISIBLE_H/2, SCREEN_VISIBLE_H/2,
    "ScreenHole"
)
boolean_op(front_bezel, screen_hole)

screen_recess = create_block(
    -SCREEN_W/2, SCREEN_W/2,
    -5.0, 0.1, 
    -SCREEN_H/2, SCREEN_H/2,
    "ScreenRecess"
)
boolean_op(front_bezel, screen_recess)

# 组装卡扣
lip = create_block(
    -CASE_WIDTH/2 + WALL - 0.2, CASE_WIDTH/2 - WALL + 0.2,
    0, 3.0,
    -CASE_HEIGHT/2 + WALL - 0.2, CASE_HEIGHT/2 - WALL + 0.2,
    "AssemblyLip"
)
lip_inner = create_block(
    -CASE_WIDTH/2 + WALL + 2, CASE_WIDTH/2 - WALL - 2,
    -1, 4,
    -CASE_HEIGHT/2 + WALL + 2, CASE_HEIGHT/2 - WALL - 2,
    "LipInner"
)
boolean_op(lip, lip_inner)
boolean_op(front_bezel, lip, 'UNION')

# ==================== 5. 装饰与后处理 ====================
# 侧面散热孔 (Side Vents)
for i in range(3):
    z = (i - 1) * 15
    vent = create_block(
        -CASE_WIDTH/2 - 1, -CASE_WIDTH/2 + 5,
        20, 60,
        z - 2, z + 2,
        "VentL"
    )
    boolean_op(rear_shell, vent)
    
    vent2 = create_block(
        CASE_WIDTH/2 - 5, CASE_WIDTH/2 + 1,
        20, 60,
        z - 2, z + 2,
        "VentR"
    )
    boolean_op(rear_shell, vent2)

# 倒角
add_bevel(front_bezel, 1.5)
add_bevel(rear_shell, 2.0)

front_bezel.parent = rear_shell

bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.shade_smooth()

print("=" * 60)
print("复古外壳 V3 (底部走线版) 生成完毕！")
print("顶部：完全封闭 (无开孔)")
print("底部：设有大型线缆舱 (Cable Bay)，支持 USB/RJ45 及 Type-C 走线")
print("内部：顶部预留 27mm 空间，供 Type-C 线缆内部回旋")
print("=" * 60)
