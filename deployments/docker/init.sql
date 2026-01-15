-- 初始化数据库脚本

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS goback DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE goback;

-- 初始化管理员用户
INSERT INTO sys_user (username, password, nickname, email, phone, status, created_at, updated_at) 
VALUES ('admin', '$2a$10$EqKR.g1KPjY.s5Z.5P5YeOAKoHoJZqT0rVYvOHxQJ5jJTVNxQ.fZe', '超级管理员', 'admin@example.com', '13800138000', 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 初始化角色
INSERT INTO sys_role (name, code, sort, status, remark, created_at, updated_at)
VALUES 
    ('超级管理员', 'admin', 1, 1, '拥有所有权限', NOW(), NOW()),
    ('普通用户', 'user', 2, 1, '普通用户角色', NOW(), NOW())
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 初始化权限
INSERT INTO sys_permission (name, code, type, parent_id, sort, status, created_at, updated_at)
VALUES 
    ('系统管理', 'system', 'menu', 0, 1, 1, NOW(), NOW()),
    ('用户管理', 'system:user', 'menu', 1, 1, 1, NOW(), NOW()),
    ('用户查询', 'system:user:list', 'button', 2, 1, 1, NOW(), NOW()),
    ('用户新增', 'system:user:add', 'button', 2, 2, 1, NOW(), NOW()),
    ('用户修改', 'system:user:edit', 'button', 2, 3, 1, NOW(), NOW()),
    ('用户删除', 'system:user:delete', 'button', 2, 4, 1, NOW(), NOW()),
    ('角色管理', 'system:role', 'menu', 1, 2, 1, NOW(), NOW()),
    ('菜单管理', 'system:menu', 'menu', 1, 3, 1, NOW(), NOW()),
    ('日志管理', 'system:log', 'menu', 1, 4, 1, NOW(), NOW()),
    ('字典管理', 'system:dict', 'menu', 1, 5, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 初始化角色权限关联
INSERT INTO sys_role_permission (role_id, permission_id, created_at)
SELECT 1, id, NOW() FROM sys_permission
ON DUPLICATE KEY UPDATE created_at = NOW();

-- 初始化菜单
INSERT INTO sys_menu (name, parent_id, path, component, icon, sort, type, visible, status, created_at, updated_at)
VALUES 
    ('系统管理', 0, '/system', 'Layout', 'setting', 1, 'catalog', 1, 1, NOW(), NOW()),
    ('用户管理', 1, '/system/user', 'system/user/index', 'user', 1, 'menu', 1, 1, NOW(), NOW()),
    ('角色管理', 1, '/system/role', 'system/role/index', 'peoples', 2, 'menu', 1, 1, NOW(), NOW()),
    ('菜单管理', 1, '/system/menu', 'system/menu/index', 'tree-table', 3, 'menu', 1, 1, NOW(), NOW()),
    ('日志管理', 1, '/system/log', 'system/log/index', 'log', 4, 'menu', 1, 1, NOW(), NOW()),
    ('字典管理', 1, '/system/dict', 'system/dict/index', 'dict', 5, 'menu', 1, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 初始化字典类型
INSERT INTO sys_dict_type (name, code, status, remark, created_at, updated_at)
VALUES 
    ('用户状态', 'sys_user_status', 1, '用户状态列表', NOW(), NOW()),
    ('通用状态', 'sys_common_status', 1, '通用状态列表', NOW(), NOW()),
    ('性别', 'sys_user_sex', 1, '用户性别列表', NOW(), NOW())
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 初始化字典数据
INSERT INTO sys_dict_data (type_id, label, value, sort, status, created_at, updated_at)
VALUES 
    (1, '正常', '1', 1, 1, NOW(), NOW()),
    (1, '禁用', '0', 2, 1, NOW(), NOW()),
    (2, '启用', '1', 1, 1, NOW(), NOW()),
    (2, '停用', '0', 2, 1, NOW(), NOW()),
    (3, '男', '1', 1, 1, NOW(), NOW()),
    (3, '女', '2', 2, 1, NOW(), NOW()),
    (3, '未知', '0', 3, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 更新用户角色
UPDATE sys_user SET role_id = 1 WHERE username = 'admin';

-- 创建系统参数配置表
CREATE TABLE IF NOT EXISTS sys_config (
    config_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    config_name VARCHAR(100) NOT NULL COMMENT '参数名称',
    config_key VARCHAR(100) NOT NULL COMMENT '参数键名',
    config_value TEXT NOT NULL COMMENT '参数键值',
    config_type VARCHAR(1) DEFAULT 'N' COMMENT '是否内置参数（Y是/N否）',
    create_by BIGINT DEFAULT 0 COMMENT '创建者',
    remark VARCHAR(500) COMMENT '备注',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at DATETIME NULL COMMENT '删除时间',
    UNIQUE KEY idx_config_key (config_key),
    KEY idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统参数配置表';

-- 初始化系统参数配置（示例数据）
INSERT INTO sys_config (config_name, config_key, config_value, config_type, remark, created_at, updated_at)
VALUES 
    ('系统名称', 'sys.name', 'GoBack管理系统', 'Y', '系统显示名称（内置）', NOW(), NOW()),
    ('系统版本', 'sys.version', '1.0.0', 'Y', '当前系统版本号（内置）', NOW(), NOW()),
    ('是否开放注册', 'sys.registration.enabled', 'false', 'N', '控制用户注册功能开关', NOW(), NOW()),
    ('验证码有效期', 'sys.captcha.expire', '300', 'N', '验证码有效期（秒）', NOW(), NOW())
ON DUPLICATE KEY UPDATE updated_at = NOW();
