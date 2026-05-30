-- Description: MySQL demo seed data for admin
-- Rule: preserve Go bootstrap base data, only refresh demo data owned by this script

SET FOREIGN_KEY_CHECKS = 0;
START TRANSACTION;

-- Non-user domain demo data can be fully refreshed.
TRUNCATE TABLE sys_tasks;
TRUNCATE TABLE sys_login_policies;
TRUNCATE TABLE sys_dict_entry_i18n;
TRUNCATE TABLE sys_dict_entries;
TRUNCATE TABLE sys_dict_type_i18n;
TRUNCATE TABLE sys_dict_types;
TRUNCATE TABLE internal_message_categories;

-- User-domain demo data must be incremental and must not overwrite base seed data.
DELETE FROM sys_membership_roles WHERE id BETWEEN 1401 AND 1499;
DELETE FROM sys_user_roles WHERE user_id BETWEEN 1001 AND 1099;
DELETE FROM sys_user_org_units WHERE user_id BETWEEN 1001 AND 1099;
DELETE FROM sys_user_positions WHERE user_id BETWEEN 1001 AND 1099;
DELETE FROM sys_memberships WHERE id BETWEEN 1301 AND 1399;
DELETE FROM sys_user_credentials
WHERE user_id BETWEEN 1001 AND 1099
   OR identifier IN (
        'tenant_admin', 'wangli', 'chenhao', 'liuna', 'zhaoqiang', 'sunyue', 'hejun', 'gaomin', 'zhouxin', 'tangmin',
        'medical_admin', 'xurui', 'zhengyi',
        'supply_admin', 'luobin', 'xiaqing',
        'tenant@company.com', 'wangli@eastmfg.com', 'chenhao@eastmfg.com', 'liuna@eastmfg.com', 'zhaoqiang@eastmfg.com',
        'sunyue@eastmfg.com', 'hejun@eastmfg.com', 'gaomin@eastmfg.com', 'zhouxin@eastmfg.com', 'tangmin@eastmfg.com',
        'admin@xinglanmed.com', 'xurui@xinglanmed.com', 'zhengyi@xinglanmed.com',
        'admin@yuanzhousc.com', 'luobin@yuanzhousc.com', 'xiaqing@yuanzhousc.com'
   );
DELETE FROM sys_users WHERE id BETWEEN 1001 AND 1099;
DELETE FROM sys_positions WHERE id BETWEEN 1201 AND 1299;
DELETE FROM sys_org_units WHERE id BETWEEN 1101 AND 1199;
DELETE FROM sys_tenants WHERE id BETWEEN 101 AND 199 OR code IN ('east-manufacturing', 'xinglan-medical', 'yuanzhou-supply');

INSERT INTO sys_tenants(id, name, code, type, audit_status, status, admin_user_id, created_at)
VALUES
    (101, '华东智能制造集团', 'east-manufacturing', 'PAID', 'APPROVED', 'ON', 1001, NOW()),
    (102, '星岚医疗科技', 'xinglan-medical', 'PAID', 'APPROVED', 'ON', 1011, NOW()),
    (103, '远舟供应链服务', 'yuanzhou-supply', 'TRIAL', 'APPROVED', 'ON', 1014, NOW());
ALTER TABLE sys_tenants AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_tenants);

INSERT INTO sys_users (
    id, tenant_id, username, nickname, realname, email, mobile, telephone, address, region, description, gender, status, last_login_ip, created_at
)
VALUES
    (1001, 101, 'tenant_admin', '租户管理', '张管理员', 'tenant@company.com', '13810000001', '021-68880001', '上海市浦东新区张江高科园区', 'CN-SH', '集团租户管理员，负责租户初始化与权限分配', 'MALE', 'NORMAL', '192.168.10.10', NOW()),
    (1002, 101, 'wangli', '王丽', '王丽', 'wangli@eastmfg.com', '13810000002', '021-68880002', '上海市浦东新区金科路 88 号', 'CN-SH', '技术总监，统筹研发平台与架构治理', 'FEMALE', 'NORMAL', '192.168.10.21', NOW()),
    (1003, 101, 'chenhao', '陈昊', '陈昊', 'chenhao@eastmfg.com', '13810000003', '021-68880003', '上海市浦东新区碧波路 500 号', 'CN-SH', '技术部经理，负责研发团队日常管理', 'MALE', 'NORMAL', '192.168.10.22', NOW()),
    (1004, 101, 'liuna', '刘娜', '刘娜', 'liuna@eastmfg.com', '13810000004', '021-68880004', '上海市浦东新区祖冲之路 77 弄', 'CN-SH', '前端主管，负责管理后台与可视化产品体验', 'FEMALE', 'NORMAL', '192.168.10.23', NOW()),
    (1005, 101, 'zhaoqiang', '赵强', '赵强', 'zhaoqiang@eastmfg.com', '13810000005', '021-68880005', '上海市浦东新区晨晖路 1000 号', 'CN-SH', '后端主管，负责核心业务服务与网关治理', 'MALE', 'NORMAL', '192.168.10.24', NOW()),
    (1006, 101, 'sunyue', '孙悦', '孙悦', 'sunyue@eastmfg.com', '13810000006', '021-68880006', '上海市浦东新区科苑路 151 号', 'CN-SH', '前端开发工程师，负责门户与运营端页面', 'FEMALE', 'NORMAL', '192.168.10.25', NOW()),
    (1007, 101, 'hejun', '何军', '何军', 'hejun@eastmfg.com', '13810000007', '021-68880007', '上海市浦东新区纳贤路 60 号', 'CN-SH', '后端开发工程师，负责订单与审计接口', 'MALE', 'NORMAL', '192.168.10.26', NOW()),
    (1008, 101, 'gaomin', '高敏', '高敏', 'gaomin@eastmfg.com', '13810000008', '021-68880008', '上海市浦东新区盛夏路 800 弄', 'CN-SH', '测试工程师，负责回归测试与自动化测试', 'FEMALE', 'NORMAL', '192.168.10.27', NOW()),
    (1009, 101, 'zhouxin', '周鑫', '周鑫', 'zhouxin@eastmfg.com', '13810000009', '021-68880009', '上海市黄浦区延安东路 168 号', 'CN-SH', '人力总监，负责组织发展与人才体系', 'MALE', 'NORMAL', '192.168.11.10', NOW()),
    (1010, 101, 'tangmin', '唐敏', '唐敏', 'tangmin@eastmfg.com', '13810000019', '021-68880019', '广州市海珠区新港东路 1068 号', 'CN-GD', '客服一组组长，负责售后与投诉闭环', 'FEMALE', 'NORMAL', '192.168.14.10', NOW()),
    (1011, 102, 'medical_admin', '医疗管理', '顾嘉宁', 'admin@xinglanmed.com', '13920000001', '025-86660001', '南京市建邺区江东中路 369 号', 'CN-JS', '医疗科技租户管理员，负责院区数据权限配置', 'FEMALE', 'NORMAL', '172.16.20.10', NOW()),
    (1012, 102, 'xurui', '许锐', '许锐', 'xurui@xinglanmed.com', '13920000002', '025-86660002', '南京市鼓楼区中央路 188 号', 'CN-JS', '医疗产品经理，负责病案质控平台需求', 'MALE', 'NORMAL', '172.16.20.11', NOW()),
    (1013, 102, 'zhengyi', '郑毅', '郑毅', 'zhengyi@xinglanmed.com', '13920000003', '025-86660003', '苏州市工业园区星湖街 328 号', 'CN-JS', '实施顾问，负责医院上线与培训', 'FEMALE', 'NORMAL', '172.16.20.12', NOW()),
    (1014, 103, 'supply_admin', '供应链管理', '沈清', 'admin@yuanzhousc.com', '13730000001', '0571-88080001', '杭州市滨江区网商路 699 号', 'CN-ZJ', '供应链租户管理员，负责试用期租户初始化', 'FEMALE', 'NORMAL', '10.20.30.10', NOW()),
    (1015, 103, 'luobin', '罗斌', '罗斌', 'luobin@yuanzhousc.com', '13730000002', '0571-88080002', '杭州市拱墅区丰潭路 380 号', 'CN-ZJ', '仓配主管，负责仓内调度与承运商对接', 'MALE', 'PENDING', '10.20.30.11', NOW()),
    (1016, 103, 'xiaqing', '夏青', '夏青', 'xiaqing@yuanzhousc.com', '13730000003', '0571-88080003', '宁波市鄞州区中山东路 1083 号', 'CN-ZJ', '客户成功专员，负责客户培训与问题跟进', 'FEMALE', 'NORMAL', '10.20.30.12', NOW());
ALTER TABLE sys_users AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_users);

INSERT INTO sys_user_credentials (user_id, identity_type, identifier, credential_type, credential, status, is_primary, created_at)
VALUES
    (1001, 'USERNAME', 'tenant_admin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1001, 'EMAIL', 'tenant@company.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1002, 'USERNAME', 'wangli', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1002, 'EMAIL', 'wangli@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1003, 'USERNAME', 'chenhao', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1003, 'EMAIL', 'chenhao@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1004, 'USERNAME', 'liuna', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1004, 'EMAIL', 'liuna@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1005, 'USERNAME', 'zhaoqiang', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1005, 'EMAIL', 'zhaoqiang@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1006, 'USERNAME', 'sunyue', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1006, 'EMAIL', 'sunyue@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1007, 'USERNAME', 'hejun', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1007, 'EMAIL', 'hejun@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1008, 'USERNAME', 'gaomin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1008, 'EMAIL', 'gaomin@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1009, 'USERNAME', 'zhouxin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1009, 'EMAIL', 'zhouxin@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1010, 'USERNAME', 'tangmin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1010, 'EMAIL', 'tangmin@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1011, 'USERNAME', 'medical_admin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1011, 'EMAIL', 'admin@xinglanmed.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1012, 'USERNAME', 'xurui', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1012, 'EMAIL', 'xurui@xinglanmed.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1013, 'USERNAME', 'zhengyi', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1013, 'EMAIL', 'zhengyi@xinglanmed.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1014, 'USERNAME', 'supply_admin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1014, 'EMAIL', 'admin@yuanzhousc.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW()),
    (1015, 'USERNAME', 'luobin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'UNVERIFIED', true, NOW()),
    (1015, 'EMAIL', 'luobin@yuanzhousc.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'UNVERIFIED', false, NOW()),
    (1016, 'USERNAME', 'xiaqing', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, NOW()),
    (1016, 'EMAIL', 'xiaqing@yuanzhousc.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, NOW());
ALTER TABLE sys_user_credentials AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_user_credentials);

INSERT INTO sys_org_units (id, tenant_id, parent_id, type, name, code, description, path, sort_order, leader_id, status, created_at)
VALUES
    (1101, 101, NULL, 'COMPANY', '华东智能制造集团总部', 'EMFG-HQ', '集团核心管理机构', '/1101', 1, 1001, 'ON', NOW()),
    (1102, 101, 1101, 'DIVISION', '技术中心', 'EMFG-TECH', '负责平台研发与技术治理', '/1101/1102', 1, 1002, 'ON', NOW()),
    (1103, 101, 1101, 'DIVISION', '人力资源部', 'EMFG-HR', '负责组织发展与招聘体系', '/1101/1103', 2, 1009, 'ON', NOW()),
    (1104, 101, 1101, 'DEPARTMENT', '客户服务部', 'EMFG-CS', '负责售后与客户成功', '/1101/1104', 3, 1010, 'ON', NOW()),
    (1105, 101, 1102, 'DEPARTMENT', '前端研发部', 'EMFG-FE', '负责前端与可视化研发', '/1101/1102/1105', 1, 1004, 'ON', NOW()),
    (1106, 101, 1102, 'DEPARTMENT', '后端研发部', 'EMFG-BE', '负责后端服务研发', '/1101/1102/1106', 2, 1005, 'ON', NOW()),
    (1107, 102, NULL, 'COMPANY', '星岚医疗科技', 'XLMED-HQ', '医疗 SaaS 业务总部', '/1107', 1, 1011, 'ON', NOW()),
    (1108, 102, 1107, 'DEPARTMENT', '产品实施部', 'XLMED-PS', '负责医院项目交付与培训', '/1107/1108', 1, 1013, 'ON', NOW()),
    (1109, 103, NULL, 'COMPANY', '远舟供应链服务', 'YZSC-HQ', '供应链业务总部', '/1109', 1, 1014, 'ON', NOW()),
    (1110, 103, 1109, 'DEPARTMENT', '仓配运营部', 'YZSC-WMS', '负责仓配运营与客户交付', '/1109/1110', 1, 1015, 'ON', NOW());
ALTER TABLE sys_org_units AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_org_units);

INSERT INTO sys_positions (
    id, tenant_id, type, name, code, org_unit_id, reports_to_position_id, description, job_family, job_grade, level, headcount, is_key_position, status, sort_order, created_at
)
VALUES
    (1201, 101, 'LEADER', '租户管理员', 'EMFG-ADMIN', 1101, NULL, '负责租户整体管理与初始化', 'MGMT', 1, 1, 1, true, 'ON', 1, NOW()),
    (1202, 101, 'LEADER', '技术总监', 'EMFG-TECH-DIR', 1102, 1201, '负责技术战略与架构治理', 'TECH', 1, 2, 1, true, 'ON', 2, NOW()),
    (1203, 101, 'MANAGER', '前端主管', 'EMFG-FE-LEAD', 1105, 1202, '负责前端团队管理', 'TECH', 2, 3, 2, false, 'ON', 3, NOW()),
    (1204, 101, 'MANAGER', '后端主管', 'EMFG-BE-LEAD', 1106, 1202, '负责后端团队管理', 'TECH', 2, 3, 2, false, 'ON', 4, NOW()),
    (1205, 101, 'REGULAR', '前端开发工程师', 'EMFG-FE-DEV', 1105, 1203, '负责前端页面与交互实现', 'TECH', 3, 4, 4, false, 'ON', 5, NOW()),
    (1206, 101, 'REGULAR', '后端开发工程师', 'EMFG-BE-DEV', 1106, 1204, '负责后端接口与业务逻辑', 'TECH', 3, 4, 4, false, 'ON', 6, NOW()),
    (1207, 101, 'REGULAR', '测试工程师', 'EMFG-QA', 1102, 1202, '负责测试与质量保障', 'TECH', 3, 4, 2, false, 'ON', 7, NOW()),
    (1208, 101, 'LEADER', '人力总监', 'EMFG-HR-DIR', 1103, 1201, '负责组织发展与人才体系', 'HR', 1, 2, 1, true, 'ON', 8, NOW()),
    (1209, 101, 'LEADER', '客服组长', 'EMFG-CS-LEAD', 1104, 1201, '负责售后服务与工单闭环', 'CS', 2, 2, 1, false, 'ON', 9, NOW()),
    (1210, 102, 'LEADER', '医疗租户管理员', 'XLMED-ADMIN', 1107, NULL, '负责医疗租户管理', 'MGMT', 1, 1, 1, true, 'ON', 1, NOW()),
    (1211, 102, 'REGULAR', '医疗产品经理', 'XLMED-PM', 1108, 1210, '负责医疗产品需求管理', 'PRODUCT', 2, 2, 1, false, 'ON', 2, NOW()),
    (1212, 102, 'REGULAR', '实施顾问', 'XLMED-IMPL', 1108, 1210, '负责医院项目实施与培训', 'DELIVERY', 2, 2, 2, false, 'ON', 3, NOW()),
    (1213, 103, 'LEADER', '供应链租户管理员', 'YZSC-ADMIN', 1109, NULL, '负责供应链租户管理', 'MGMT', 1, 1, 1, true, 'ON', 1, NOW()),
    (1214, 103, 'MANAGER', '仓配主管', 'YZSC-WMS-LEAD', 1110, 1213, '负责仓配运营与调度', 'OPS', 2, 2, 1, false, 'ON', 2, NOW()),
    (1215, 103, 'REGULAR', '客户成功专员', 'YZSC-CSM', 1110, 1214, '负责客户培训与问题跟进', 'CS', 3, 3, 2, false, 'ON', 3, NOW());
ALTER TABLE sys_positions AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_positions);

INSERT INTO sys_memberships (id, tenant_id, user_id, org_unit_id, position_id, role_id, is_primary, status)
VALUES
    (1301, 101, 1001, 1101, 1201, NULL, true, 'ACTIVE'),
    (1302, 101, 1002, 1102, 1202, NULL, true, 'ACTIVE'),
    (1303, 101, 1003, 1102, 1202, NULL, true, 'ACTIVE'),
    (1304, 101, 1004, 1105, 1203, NULL, true, 'ACTIVE'),
    (1305, 101, 1005, 1106, 1204, NULL, true, 'ACTIVE'),
    (1306, 101, 1006, 1105, 1205, NULL, true, 'ACTIVE'),
    (1307, 101, 1007, 1106, 1206, NULL, true, 'ACTIVE'),
    (1308, 101, 1008, 1102, 1207, NULL, true, 'ACTIVE'),
    (1309, 101, 1009, 1103, 1208, NULL, true, 'ACTIVE'),
    (1310, 101, 1010, 1104, 1209, NULL, true, 'ACTIVE'),
    (1311, 102, 1011, 1107, 1210, NULL, true, 'ACTIVE'),
    (1312, 102, 1012, 1108, 1211, NULL, true, 'ACTIVE'),
    (1313, 102, 1013, 1108, 1212, NULL, true, 'ACTIVE'),
    (1314, 103, 1014, 1109, 1213, NULL, true, 'ACTIVE'),
    (1315, 103, 1015, 1110, 1214, NULL, true, 'PENDING'),
    (1316, 103, 1016, 1110, 1215, NULL, true, 'ACTIVE');
ALTER TABLE sys_memberships AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_memberships);

INSERT INTO sys_user_org_units (
    tenant_id, remark, user_id, org_unit_id, position_id, assigned_at, assigned_by, is_primary, status, created_at, updated_at
)
SELECT tenant_id, 'demo seed', user_id, org_unit_id, position_id, NOW(), 0, is_primary, 'ACTIVE', NOW(), NOW()
FROM sys_memberships
WHERE id BETWEEN 1301 AND 1399 AND org_unit_id IS NOT NULL;
ALTER TABLE sys_user_org_units AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_user_org_units);

INSERT INTO sys_user_positions (
    tenant_id, remark, user_id, position_id, is_primary, assigned_at, assigned_by, status, created_at, updated_at
)
SELECT tenant_id, 'demo seed', user_id, position_id, is_primary, NOW(), 0, 'ACTIVE', NOW(), NOW()
FROM sys_memberships
WHERE id BETWEEN 1301 AND 1399 AND position_id IS NOT NULL;
ALTER TABLE sys_user_positions AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_user_positions);
ALTER TABLE sys_user_roles AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_user_roles);
ALTER TABLE sys_membership_roles AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_membership_roles);

INSERT INTO sys_tasks(type, type_name, task_payload, cron_spec, enable, created_at)
VALUES
    ('PERIODIC', 'backup', '{ \"name\": \"demo-backup\" }', '0 * * * *', true, NOW());
ALTER TABLE sys_tasks AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_tasks);

INSERT INTO sys_login_policies(id, target_id, type, method, value, reason, created_at)
VALUES
    (1, 1, 'BLACKLIST', 'IP', '127.0.0.2', '演示环境黑名单示例', NOW()),
    (2, 1, 'WHITELIST', 'MAC', '00:1B:44:11:3A:B7', '演示环境白名单示例', NOW());
ALTER TABLE sys_login_policies AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_login_policies);

INSERT INTO sys_dict_types(id, type_code, type_name, sort_order, description, is_enabled, created_at)
VALUES
    (1, 'USER_STATUS', '用户状态', 10, '系统用户状态管理', true, NOW()),
    (2, 'DEVICE_TYPE', '设备类型', 20, '平台接入设备分类', true, NOW()),
    (3, 'ORDER_STATUS', '订单状态', 30, '订单全生命周期状态', true, NOW()),
    (4, 'GENDER', '性别', 40, '用户性别枚举', true, NOW()),
    (5, 'PAYMENT_METHOD', '支付方式', 50, '系统支持的支付方式', true, NOW());
ALTER TABLE sys_dict_types AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_dict_types);

INSERT INTO sys_dict_type_i18n(dict_type_id, language_code, type_name, description, tenant_id, created_at, updated_at)
VALUES
    (1, 'zh-CN', '用户状态', '系统用户状态管理', 0, NOW(), NOW()),
    (2, 'zh-CN', '设备类型', '平台接入设备分类', 0, NOW(), NOW()),
    (3, 'zh-CN', '订单状态', '订单全生命周期状态', 0, NOW(), NOW()),
    (4, 'zh-CN', '性别', '用户性别枚举', 0, NOW(), NOW()),
    (5, 'zh-CN', '支付方式', '系统支持的支付方式', 0, NOW(), NOW()),
    (1, 'en-US', 'User Status', 'System user status management', 0, NOW(), NOW()),
    (2, 'en-US', 'Device Type', 'Device categories connected to the platform', 0, NOW(), NOW()),
    (3, 'en-US', 'Order Status', 'Full lifecycle statuses for orders', 0, NOW(), NOW()),
    (4, 'en-US', 'Gender', 'User gender enumeration', 0, NOW(), NOW()),
    (5, 'en-US', 'Payment Method', 'Supported payment channels', 0, NOW(), NOW());
ALTER TABLE sys_dict_type_i18n AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_dict_type_i18n);

INSERT INTO sys_dict_entries(id, type_id, entry_value, entry_label, numeric_value, sort_order, description, is_enabled, created_at)
VALUES
    (1, 1, 'NORMAL', '正常', 1, 1, '用户可以正常登录和操作', true, NOW()),
    (2, 1, 'FROZEN', '冻结', 2, 2, '用户被暂时冻结', true, NOW()),
    (3, 1, 'CANCELED', '注销', 3, 3, '用户已注销', true, NOW()),
    (4, 2, 'TEMP_SENSOR', '温湿度传感器', 101, 1, '支持温湿度采集', true, NOW()),
    (5, 2, 'CURRENT_METER', '电流仪表', 102, 2, '支持交流和直流电流测量', true, NOW()),
    (6, 2, 'GAS_DETECTOR', '气体探测器', 103, 3, '等待后续硬件适配', false, NOW()),
    (7, 3, 'PENDING', '待支付', 1, 1, '订单待支付', true, NOW()),
    (8, 3, 'PAID', '已支付', 2, 2, '订单已支付', true, NOW()),
    (9, 3, 'SHIPPED', '已发货', 3, 3, '订单已发货', true, NOW()),
    (10, 3, 'COMPLETED', '已完成', 4, 4, '订单已完成', true, NOW()),
    (11, 3, 'CANCELED', '已取消', 5, 5, '订单已取消', true, NOW()),
    (12, 4, 'MALE', '男', 1, 1, '', true, NOW()),
    (13, 4, 'FEMALE', '女', 2, 2, '', true, NOW()),
    (14, 4, 'UNKNOWN', '未知', 0, 3, '默认值', true, NOW()),
    (15, 5, 'ALIPAY', '支付宝', 1, 1, '支持花呗和余额宝', true, NOW()),
    (16, 5, 'WECHAT', '微信支付', 2, 2, '需要绑定微信', true, NOW()),
    (17, 5, 'UNIONPAY', '银联支付', 3, 3, '支持信用卡和借记卡', true, NOW()),
    (18, 5, 'CASH', '现金支付', 4, 4, '线下支付，已废弃', false, NOW());
ALTER TABLE sys_dict_entries AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_dict_entries);

INSERT INTO sys_dict_entry_i18n(entry_id, language_code, entry_label, description, sort_order, tenant_id, created_at, updated_at)
VALUES
    (1, 'zh-CN', '正常', '用户可以正常登录和操作', 1, 0, NOW(), NOW()),
    (2, 'zh-CN', '冻结', '用户被暂时冻结', 2, 0, NOW(), NOW()),
    (3, 'zh-CN', '注销', '用户已注销', 3, 0, NOW(), NOW()),
    (4, 'zh-CN', '温湿度传感器', '支持温湿度采集', 1, 0, NOW(), NOW()),
    (5, 'zh-CN', '电流仪表', '支持交流和直流电流测量', 2, 0, NOW(), NOW()),
    (6, 'zh-CN', '气体探测器', '等待后续硬件适配', 3, 0, NOW(), NOW()),
    (7, 'zh-CN', '待支付', '订单待支付', 1, 0, NOW(), NOW()),
    (8, 'zh-CN', '已支付', '订单已支付', 2, 0, NOW(), NOW()),
    (9, 'zh-CN', '已发货', '订单已发货', 3, 0, NOW(), NOW()),
    (10, 'zh-CN', '已完成', '订单已完成', 4, 0, NOW(), NOW()),
    (11, 'zh-CN', '已取消', '订单已取消', 5, 0, NOW(), NOW()),
    (12, 'zh-CN', '男', '', 1, 0, NOW(), NOW()),
    (13, 'zh-CN', '女', '', 2, 0, NOW(), NOW()),
    (14, 'zh-CN', '未知', '默认值', 3, 0, NOW(), NOW()),
    (15, 'zh-CN', '支付宝', '支持花呗和余额宝', 1, 0, NOW(), NOW()),
    (16, 'zh-CN', '微信支付', '需要绑定微信', 2, 0, NOW(), NOW()),
    (17, 'zh-CN', '银联支付', '支持信用卡和借记卡', 3, 0, NOW(), NOW()),
    (18, 'zh-CN', '现金支付', '线下支付，已废弃', 4, 0, NOW(), NOW()),
    (1, 'en-US', 'Normal', 'User can log in and operate normally', 1, 0, NOW(), NOW()),
    (2, 'en-US', 'Frozen', 'User is temporarily frozen', 2, 0, NOW(), NOW()),
    (3, 'en-US', 'Canceled', 'User has been canceled', 3, 0, NOW(), NOW()),
    (4, 'en-US', 'Temperature & Humidity Sensor', 'Supports temperature and humidity collection', 1, 0, NOW(), NOW()),
    (5, 'en-US', 'Current Meter', 'Supports AC and DC current measurement', 2, 0, NOW(), NOW()),
    (6, 'en-US', 'Gas Detector', 'Waiting for later hardware integration', 3, 0, NOW(), NOW()),
    (7, 'en-US', 'Pending Payment', 'Order is pending payment', 1, 0, NOW(), NOW()),
    (8, 'en-US', 'Paid', 'Order has been paid', 2, 0, NOW(), NOW()),
    (9, 'en-US', 'Shipped', 'Order has been shipped', 3, 0, NOW(), NOW()),
    (10, 'en-US', 'Completed', 'Order has been completed', 4, 0, NOW(), NOW()),
    (11, 'en-US', 'Canceled', 'Order has been canceled', 5, 0, NOW(), NOW()),
    (12, 'en-US', 'Male', '', 1, 0, NOW(), NOW()),
    (13, 'en-US', 'Female', '', 2, 0, NOW(), NOW()),
    (14, 'en-US', 'Unknown', 'Default value', 3, 0, NOW(), NOW()),
    (15, 'en-US', 'Alipay', 'Supports Huabei and Yu''ebao', 1, 0, NOW(), NOW()),
    (16, 'en-US', 'WeChat Pay', 'Requires WeChat binding', 2, 0, NOW(), NOW()),
    (17, 'en-US', 'UnionPay', 'Supports credit and debit cards', 3, 0, NOW(), NOW()),
    (18, 'en-US', 'Cash', 'Offline payment, deprecated', 4, 0, NOW(), NOW());
ALTER TABLE sys_dict_entry_i18n AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM sys_dict_entry_i18n);

INSERT INTO internal_message_categories (id, code, name, remark, sort_order, is_enabled, created_at)
VALUES
    (1, 'order', '订单通知', '包含订单支付、发货、退款等全流程通知', 1, true, NOW()),
    (101, 'order_paid', '支付成功', '订单支付完成时触发的通知', 2, true, NOW()),
    (102, 'order_unpaid', '支付超时', '订单未在规定时间内支付的提醒', 3, true, NOW()),
    (103, 'order_shipped', '已发货', '商家发货后通知用户', 4, true, NOW()),
    (104, 'order_refunded', '已退款', '订单退款流程完成的通知', 5, true, NOW()),
    (2, 'system', '系统通知', '系统公告、维护提醒、版本更新等平台级通知', 6, true, NOW()),
    (201, 'system_announcement', '系统公告', '平台规则更新、重要通知等', 7, true, NOW()),
    (202, 'system_maintenance', '维护通知', '系统计划维护的时间提醒', 8, true, NOW()),
    (203, 'system_upgrade', '版本更新', '客户端或功能升级提示', 9, true, NOW()),
    (3, 'activity', '活动通知', '营销活动报名、开始、结束等提醒', 10, true, NOW()),
    (301, 'activity_signup', '报名成功', '用户报名活动后的确认通知', 11, true, NOW()),
    (302, 'activity_start', '活动开始', '活动即将开始的倒计时提醒', 12, true, NOW()),
    (303, 'activity_end', '活动结束', '活动结束及结果公示通知', 13, true, NOW()),
    (4, 'user', '用户通知', '账号安全、资料变更、权限调整等通知', 14, true, NOW()),
    (401, 'user_login_abnormal', '异地登录', '账号在陌生设备登录的安全提醒', 15, true, NOW()),
    (402, 'user_profile_updated', '资料变更', '用户手机号、邮箱等信息修改后的通知', 16, true, NOW()),
    (403, 'user_permission_changed', '权限变更', '账号角色或功能权限调整通知', 17, true, NOW());
ALTER TABLE internal_message_categories AUTO_INCREMENT = (SELECT COALESCE(MAX(id) + 1, 1) FROM internal_message_categories);

COMMIT;
SET FOREIGN_KEY_CHECKS = 1;

SELECT 'MySQL demo seed initialized successfully' AS result;
