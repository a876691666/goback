INSERT INTO sys_config (config_name, config_key, config_value, config_type, remark, created_at, updated_at) 
VALUES ('系统名称', 'sys.name', 'GoBack管理系统', 'Y', '系统显示名称', datetime('now'), datetime('now'));

INSERT INTO sys_config (config_name, config_key, config_value, config_type, remark, created_at, updated_at) 
VALUES ('系统版本', 'sys.version', '1.0.0', 'Y', '当前系统版本号', datetime('now'), datetime('now'));

INSERT INTO sys_config (config_name, config_key, config_value, config_type, remark, created_at, updated_at) 
VALUES ('是否开放注册', 'sys.registration.enabled', 'false', 'N', '控制用户注册功能开关', datetime('now'), datetime('now'));
