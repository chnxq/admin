BEGIN;

SET LOCAL search_path = public, pg_catalog;

TRUNCATE TABLE public.sys_tasks RESTART IDENTITY CASCADE;
TRUNCATE TABLE public.sys_login_policies RESTART IDENTITY CASCADE;
TRUNCATE TABLE public.sys_dict_label_i18n RESTART IDENTITY CASCADE;
TRUNCATE TABLE public.sys_dict_labels RESTART IDENTITY CASCADE;
TRUNCATE TABLE public.sys_dict_category_i18n RESTART IDENTITY CASCADE;
TRUNCATE TABLE public.sys_dict_categories RESTART IDENTITY CASCADE;
TRUNCATE TABLE public.internal_message_categories RESTART IDENTITY CASCADE;

DELETE FROM public.sys_membership_roles WHERE id BETWEEN 1401 AND 1499;
DELETE FROM public.sys_user_roles WHERE user_id BETWEEN 1001 AND 1099;
DELETE FROM public.sys_user_org_units WHERE user_id BETWEEN 1001 AND 1099;
DELETE FROM public.sys_user_positions WHERE user_id BETWEEN 1001 AND 1099;
DELETE FROM public.sys_memberships WHERE id BETWEEN 1301 AND 1399;
DELETE FROM public.sys_user_credentials
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
DELETE FROM public.sys_users WHERE id BETWEEN 1001 AND 1099;
DELETE FROM public.sys_positions WHERE id BETWEEN 1201 AND 1299;
DELETE FROM public.sys_org_units WHERE id BETWEEN 1101 AND 1199;
DELETE FROM public.sys_tenants WHERE id BETWEEN 101 AND 199 OR code IN ('east-manufacturing', 'xinglan-medical', 'yuanzhou-supply');

INSERT INTO public.sys_tenants(id, name, code, type, audit_status, status, admin_user_id, created_at)
VALUES
    (101, '华东智能制造集团', 'east-manufacturing', 'PAID', 'APPROVED', 'ON', 1001, now()),
    (102, '星岚医疗科技', 'xinglan-medical', 'PAID', 'APPROVED', 'ON', 1011, now()),
    (103, '远舟供应链服务', 'yuanzhou-supply', 'TRIAL', 'APPROVED', 'ON', 1014, now());
SELECT setval('sys_tenants_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_tenants));

INSERT INTO public.sys_users (
    id, tenant_id, username, nickname, realname, email, mobile, telephone, address, region, description, gender, status, last_login_ip, created_at
)
VALUES
    (1001, 101, 'tenant_admin', '租户管理', '张管理员', 'tenant@company.com', '13810000001', '021-68880001', '上海市浦东新区张江高科园区', 'CN-SH', '集团租户管理员，负责租户初始化与权限分配', 'MALE', 'NORMAL', '192.168.10.10', now()),
    (1002, 101, 'wangli', '王丽', '王丽', 'wangli@eastmfg.com', '13810000002', '021-68880002', '上海市浦东新区金科路 88 号', 'CN-SH', '技术总监，统筹研发平台与架构治理', 'FEMALE', 'NORMAL', '192.168.10.21', now()),
    (1003, 101, 'chenhao', '陈昊', '陈昊', 'chenhao@eastmfg.com', '13810000003', '021-68880003', '上海市浦东新区碧波路 500 号', 'CN-SH', '技术部经理，负责研发团队日常管理', 'MALE', 'NORMAL', '192.168.10.22', now()),
    (1004, 101, 'liuna', '刘娜', '刘娜', 'liuna@eastmfg.com', '13810000004', '021-68880004', '上海市浦东新区祖冲之路 77 弄', 'CN-SH', '前端主管，负责管理后台与可视化产品体验', 'FEMALE', 'NORMAL', '192.168.10.23', now()),
    (1005, 101, 'zhaoqiang', '赵强', '赵强', 'zhaoqiang@eastmfg.com', '13810000005', '021-68880005', '上海市浦东新区晨晖路 1000 号', 'CN-SH', '后端主管，负责核心业务服务与网关治理', 'MALE', 'NORMAL', '192.168.10.24', now()),
    (1006, 101, 'sunyue', '孙悦', '孙悦', 'sunyue@eastmfg.com', '13810000006', '021-68880006', '上海市浦东新区科苑路 151 号', 'CN-SH', '前端开发工程师，负责门户与运营端页面', 'FEMALE', 'NORMAL', '192.168.10.25', now()),
    (1007, 101, 'hejun', '何军', '何军', 'hejun@eastmfg.com', '13810000007', '021-68880007', '上海市浦东新区纳贤路 60 号', 'CN-SH', '后端开发工程师，负责订单与审计接口', 'MALE', 'NORMAL', '192.168.10.26', now()),
    (1008, 101, 'gaomin', '高敏', '高敏', 'gaomin@eastmfg.com', '13810000008', '021-68880008', '上海市浦东新区盛夏路 800 弄', 'CN-SH', '测试工程师，负责回归测试与自动化测试', 'FEMALE', 'NORMAL', '192.168.10.27', now()),
    (1009, 101, 'zhouxin', '周鑫', '周鑫', 'zhouxin@eastmfg.com', '13810000009', '021-68880009', '上海市黄浦区延安东路 168 号', 'CN-SH', '人力总监，负责组织发展与人才体系', 'MALE', 'NORMAL', '192.168.11.10', now()),
    (1010, 101, 'tangmin', '唐敏', '唐敏', 'tangmin@eastmfg.com', '13810000019', '021-68880019', '广州市海珠区新港东路 1068 号', 'CN-GD', '客服一组组长，负责售后与投诉闭环', 'FEMALE', 'NORMAL', '192.168.14.10', now()),
    (1011, 102, 'medical_admin', '医疗管理', '顾嘉宁', 'admin@xinglanmed.com', '13920000001', '025-86660001', '南京市建邺区江东中路 369 号', 'CN-JS', '医疗科技租户管理员，负责院区数据权限配置', 'FEMALE', 'NORMAL', '172.16.20.10', now()),
    (1012, 102, 'xurui', '许锐', '许锐', 'xurui@xinglanmed.com', '13920000002', '025-86660002', '南京市鼓楼区中央路 188 号', 'CN-JS', '医疗产品经理，负责病案质控平台需求', 'MALE', 'NORMAL', '172.16.20.11', now()),
    (1013, 102, 'zhengyi', '郑毅', '郑毅', 'zhengyi@xinglanmed.com', '13920000003', '025-86660003', '苏州市工业园区星湖街 328 号', 'CN-JS', '实施顾问，负责医院上线与培训', 'FEMALE', 'NORMAL', '172.16.20.12', now()),
    (1014, 103, 'supply_admin', '供应链管理', '沈清', 'admin@yuanzhousc.com', '13730000001', '0571-88080001', '杭州市滨江区网商路 699 号', 'CN-ZJ', '供应链租户管理员，负责试用期租户初始化', 'FEMALE', 'NORMAL', '10.20.30.10', now()),
    (1015, 103, 'luobin', '罗斌', '罗斌', 'luobin@yuanzhousc.com', '13730000002', '0571-88080002', '杭州市拱墅区丰潭路 380 号', 'CN-ZJ', '仓配主管，负责仓内调度与承运商对接', 'MALE', 'PENDING', '10.20.30.11', now()),
    (1016, 103, 'xiaqing', '夏青', '夏青', 'xiaqing@yuanzhousc.com', '13730000003', '0571-88080003', '宁波市鄞州区中山东路 1083 号', 'CN-ZJ', '客户成功专员，负责客户培训与问题跟进', 'FEMALE', 'NORMAL', '10.20.30.12', now());
SELECT setval('sys_users_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_users));

INSERT INTO public.sys_user_credentials (user_id, identity_type, identifier, credential_type, credential, status, is_primary, created_at)
VALUES
    (1001, 'USERNAME', 'tenant_admin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1001, 'EMAIL', 'tenant@company.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1002, 'USERNAME', 'wangli', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1002, 'EMAIL', 'wangli@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1003, 'USERNAME', 'chenhao', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1003, 'EMAIL', 'chenhao@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1004, 'USERNAME', 'liuna', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1004, 'EMAIL', 'liuna@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1005, 'USERNAME', 'zhaoqiang', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1005, 'EMAIL', 'zhaoqiang@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1006, 'USERNAME', 'sunyue', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1006, 'EMAIL', 'sunyue@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1007, 'USERNAME', 'hejun', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1007, 'EMAIL', 'hejun@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1008, 'USERNAME', 'gaomin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1008, 'EMAIL', 'gaomin@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1009, 'USERNAME', 'zhouxin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1009, 'EMAIL', 'zhouxin@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1010, 'USERNAME', 'tangmin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1010, 'EMAIL', 'tangmin@eastmfg.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1011, 'USERNAME', 'medical_admin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1011, 'EMAIL', 'admin@xinglanmed.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1012, 'USERNAME', 'xurui', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1012, 'EMAIL', 'xurui@xinglanmed.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1013, 'USERNAME', 'zhengyi', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1013, 'EMAIL', 'zhengyi@xinglanmed.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1014, 'USERNAME', 'supply_admin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1014, 'EMAIL', 'admin@yuanzhousc.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),
    (1015, 'USERNAME', 'luobin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'UNVERIFIED', true, now()),
    (1015, 'EMAIL', 'luobin@yuanzhousc.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'UNVERIFIED', false, now()),
    (1016, 'USERNAME', 'xiaqing', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1016, 'EMAIL', 'xiaqing@yuanzhousc.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now());
SELECT setval('sys_user_credentials_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_user_credentials));

INSERT INTO public.sys_org_units (id, tenant_id, parent_id, type, name, code, description, path, sort_order, leader_id, status, created_at)
VALUES
    (1101, 101, NULL, 'COMPANY', '华东智能制造集团总部', 'EMFG-HQ', '集团核心管理机构', '/1101', 1, 1001, 'ON', now()),
    (1102, 101, 1101, 'DIVISION', '技术中心', 'EMFG-TECH', '负责平台研发与技术治理', '/1101/1102', 1, 1002, 'ON', now()),
    (1103, 101, 1101, 'DIVISION', '人力资源部', 'EMFG-HR', '负责组织发展与招聘体系', '/1101/1103', 2, 1009, 'ON', now()),
    (1104, 101, 1101, 'DEPARTMENT', '客户服务部', 'EMFG-CS', '负责售后与客户成功', '/1101/1104', 3, 1010, 'ON', now()),
    (1105, 101, 1102, 'DEPARTMENT', '前端研发部', 'EMFG-FE', '负责前端与可视化研发', '/1101/1102/1105', 1, 1004, 'ON', now()),
    (1106, 101, 1102, 'DEPARTMENT', '后端研发部', 'EMFG-BE', '负责后端服务研发', '/1101/1102/1106', 2, 1005, 'ON', now()),
    (1107, 102, NULL, 'COMPANY', '星岚医疗科技', 'XLMED-HQ', '医疗 SaaS 业务总部', '/1107', 1, 1011, 'ON', now()),
    (1108, 102, 1107, 'DEPARTMENT', '产品实施部', 'XLMED-PS', '负责医院项目交付与培训', '/1107/1108', 1, 1013, 'ON', now()),
    (1109, 103, NULL, 'COMPANY', '远舟供应链服务', 'YZSC-HQ', '供应链业务总部', '/1109', 1, 1014, 'ON', now()),
    (1110, 103, 1109, 'DEPARTMENT', '仓配运营部', 'YZSC-WMS', '负责仓配运营与客户交付', '/1109/1110', 1, 1015, 'ON', now());
SELECT setval('sys_org_units_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_org_units));

INSERT INTO public.sys_positions (
    id, tenant_id, type, name, code, org_unit_id, reports_to_position_id, description, job_family, job_grade, level, headcount, is_key_position, status, sort_order, created_at
)
VALUES
    (1201, 101, 'LEADER', '租户管理员', 'EMFG-ADMIN', 1101, NULL, '负责租户整体管理与初始化', 'MGMT', 1, 1, 1, true, 'ON', 1, now()),
    (1202, 101, 'LEADER', '技术总监', 'EMFG-TECH-DIR', 1102, 1201, '负责技术战略与架构治理', 'TECH', 1, 2, 1, true, 'ON', 2, now()),
    (1203, 101, 'MANAGER', '前端主管', 'EMFG-FE-LEAD', 1105, 1202, '负责前端团队管理', 'TECH', 2, 3, 2, false, 'ON', 3, now()),
    (1204, 101, 'MANAGER', '后端主管', 'EMFG-BE-LEAD', 1106, 1202, '负责后端团队管理', 'TECH', 2, 3, 2, false, 'ON', 4, now()),
    (1205, 101, 'REGULAR', '前端开发工程师', 'EMFG-FE-DEV', 1105, 1203, '负责前端页面与交互实现', 'TECH', 3, 4, 4, false, 'ON', 5, now()),
    (1206, 101, 'REGULAR', '后端开发工程师', 'EMFG-BE-DEV', 1106, 1204, '负责后端接口与业务逻辑', 'TECH', 3, 4, 4, false, 'ON', 6, now()),
    (1207, 101, 'REGULAR', '测试工程师', 'EMFG-QA', 1102, 1202, '负责测试与质量保障', 'TECH', 3, 4, 2, false, 'ON', 7, now()),
    (1208, 101, 'LEADER', '人力总监', 'EMFG-HR-DIR', 1103, 1201, '负责组织发展与人才体系', 'HR', 1, 2, 1, true, 'ON', 8, now()),
    (1209, 101, 'LEADER', '客服组长', 'EMFG-CS-LEAD', 1104, 1201, '负责售后服务与工单闭环', 'CS', 2, 2, 1, false, 'ON', 9, now()),
    (1210, 102, 'LEADER', '医疗租户管理员', 'XLMED-ADMIN', 1107, NULL, '负责医疗租户管理', 'MGMT', 1, 1, 1, true, 'ON', 1, now()),
    (1211, 102, 'REGULAR', '医疗产品经理', 'XLMED-PM', 1108, 1210, '负责医疗产品需求管理', 'PRODUCT', 2, 2, 1, false, 'ON', 2, now()),
    (1212, 102, 'REGULAR', '实施顾问', 'XLMED-IMPL', 1108, 1210, '负责医院项目实施与培训', 'DELIVERY', 2, 2, 2, false, 'ON', 3, now()),
    (1213, 103, 'LEADER', '供应链租户管理员', 'YZSC-ADMIN', 1109, NULL, '负责供应链租户管理', 'MGMT', 1, 1, 1, true, 'ON', 1, now()),
    (1214, 103, 'MANAGER', '仓配主管', 'YZSC-WMS-LEAD', 1110, 1213, '负责仓配运营与调度', 'OPS', 2, 2, 1, false, 'ON', 2, now()),
    (1215, 103, 'REGULAR', '客户成功专员', 'YZSC-CSM', 1110, 1214, '负责客户培训与问题跟进', 'CS', 3, 3, 2, false, 'ON', 3, now());
SELECT setval('sys_positions_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_positions));

INSERT INTO public.sys_memberships (id, tenant_id, user_id, org_unit_id, position_id, role_id, is_primary, status)
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
SELECT setval('sys_memberships_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_memberships));

INSERT INTO public.sys_user_org_units (
    tenant_id, remark, user_id, org_unit_id, position_id, assigned_at, assigned_by, is_primary, status, created_at, updated_at
)
SELECT tenant_id, 'demo seed', user_id, org_unit_id, position_id, now(), 0, is_primary, 'ACTIVE', now(), now()
FROM public.sys_memberships
WHERE id BETWEEN 1301 AND 1399 AND org_unit_id IS NOT NULL;
SELECT setval('sys_user_org_units_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_user_org_units));

INSERT INTO public.sys_user_positions (
    tenant_id, remark, user_id, position_id, is_primary, assigned_at, assigned_by, status, created_at, updated_at
)
SELECT tenant_id, 'demo seed', user_id, position_id, is_primary, now(), 0, 'ACTIVE', now(), now()
FROM public.sys_memberships
WHERE id BETWEEN 1301 AND 1399 AND position_id IS NOT NULL;
SELECT setval('sys_user_positions_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_user_positions));
SELECT setval('sys_user_roles_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_user_roles));
SELECT setval('sys_membership_roles_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_membership_roles));

INSERT INTO public.sys_tasks(type, type_name, task_payload, cron_spec, enable, created_at)
VALUES
    ('PERIODIC', 'backup', '{ "name": "demo-backup" }', '0 * * * *', true, now());
SELECT setval('sys_tasks_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_tasks));

INSERT INTO public.sys_login_policies(id, target_id, type, method, value, reason, created_at)
VALUES
    (1, 1, 'BLACKLIST', 'IP', '127.0.0.2', '演示环境黑名单示例', now()),
    (2, 1, 'WHITELIST', 'MAC', '00:1B:44:11:3A:B7', '演示环境白名单示例', now());
SELECT setval('sys_login_policies_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_login_policies));

INSERT INTO public.sys_dict_categories(
    id, parent_id, path, category_key, category_name, category_level, scene, is_builtin, is_enabled, sort_order, tenant_id, description, created_at, updated_at
)
VALUES
    (1, NULL, '/', 'page', '页面', 'ROOT', 'PAGE', true, true, 10, 0, '页面类国际化分类根节点', now(), now()),
    (2, 1, '/1/', 'page.user_management', '用户管理页面', 'CHILD', 'PAGE', true, true, 11, 0, '用户与组织相关页面', now(), now()),
    (3, 1, '/1/', 'page.device_management', '设备管理页面', 'CHILD', 'PAGE', true, true, 12, 0, '设备与物联相关页面', now(), now()),
    (4, NULL, '/', 'menu', '菜单', 'ROOT', 'MENU', true, true, 20, 0, '菜单类国际化分类根节点', now(), now()),
    (5, 4, '/4/', 'menu.platform', '平台菜单', 'CHILD', 'MENU', true, true, 21, 0, '后台平台菜单项', now(), now()),
    (6, NULL, '/', 'prompt', '提示', 'ROOT', 'PROMPT', true, true, 30, 0, '提示语国际化分类根节点', now(), now()),
    (7, 6, '/6/', 'prompt.common', '通用提示', 'CHILD', 'PROMPT', true, true, 31, 0, '通用成功/确认类提示', now(), now()),
    (8, NULL, '/', 'device', '设备', 'ROOT', 'DEVICE', true, true, 40, 0, '设备业务标签分类根节点', now(), now()),
    (9, 8, '/8/', 'device.type', '设备类型', 'CHILD', 'DEVICE', true, true, 41, 0, '设备类型标签', now(), now());
SELECT setval('sys_dict_categories_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_dict_categories));

INSERT INTO public.sys_dict_category_i18n (
    category_id, language_code, display_name, description, tenant_id, created_at, updated_at
)
VALUES
    (1, 'zh-CN', '页面', '页面类国际化分类根节点', 0, now(), now()),
    (2, 'zh-CN', '用户管理页面', '用户与组织相关页面', 0, now(), now()),
    (3, 'zh-CN', '设备管理页面', '设备与物联相关页面', 0, now(), now()),
    (4, 'zh-CN', '菜单', '菜单类国际化分类根节点', 0, now(), now()),
    (5, 'zh-CN', '平台菜单', '后台平台菜单项', 0, now(), now()),
    (6, 'zh-CN', '提示', '提示语国际化分类根节点', 0, now(), now()),
    (7, 'zh-CN', '通用提示', '通用成功/确认类提示', 0, now(), now()),
    (8, 'zh-CN', '设备', '设备业务标签分类根节点', 0, now(), now()),
    (9, 'zh-CN', '设备类型', '设备类型标签', 0, now(), now()),
    (1, 'en-US', 'Page', 'Root category for page translations', 0, now(), now()),
    (2, 'en-US', 'User Management Pages', 'Pages for user and organization management', 0, now(), now()),
    (3, 'en-US', 'Device Management Pages', 'Pages for device and IoT management', 0, now(), now()),
    (4, 'en-US', 'Menu', 'Root category for menu translations', 0, now(), now()),
    (5, 'en-US', 'Platform Menu', 'Back-office platform menu items', 0, now(), now()),
    (6, 'en-US', 'Prompt', 'Root category for prompt translations', 0, now(), now()),
    (7, 'en-US', 'Common Prompt', 'Shared success and confirmation prompts', 0, now(), now()),
    (8, 'en-US', 'Device', 'Root category for device business labels', 0, now(), now()),
    (9, 'en-US', 'Device Type', 'Device type labels', 0, now(), now());
SELECT setval('sys_dict_category_i18n_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_dict_category_i18n));

INSERT INTO public.sys_dict_labels (
    id, category_id, label_key, label_code, label_kind, default_text, payload_json, is_builtin, is_enabled, status, sort_order, tenant_id, description, created_at, updated_at
)
VALUES
    (1, 2, 'page.user.list.title', 'PAGE_USER_LIST_TITLE', 'TEXT', '用户列表', '{"module":"user","view":"list"}'::jsonb, true, true, 'ON', 10, 0, '用户列表页标题', now(), now()),
    (2, 2, 'page.user.detail.title', 'PAGE_USER_DETAIL_TITLE', 'TEXT', '用户详情', '{"module":"user","view":"detail"}'::jsonb, true, true, 'ON', 11, 0, '用户详情页标题', now(), now()),
    (3, 5, 'menu.system.user', 'MENU_SYSTEM_USER', 'MENU', '用户管理', '{"icon":"mdi:account-cog","route":"/system/user"}'::jsonb, true, true, 'ON', 20, 0, '系统菜单-用户管理', now(), now()),
    (4, 5, 'menu.system.role', 'MENU_SYSTEM_ROLE', 'MENU', '角色管理', '{"icon":"mdi:shield-account","route":"/system/role"}'::jsonb, true, true, 'ON', 21, 0, '系统菜单-角色管理', now(), now()),
    (5, 7, 'prompt.common.save_success', 'PROMPT_COMMON_SAVE_SUCCESS', 'MESSAGE', '保存成功', '{"tone":"success"}'::jsonb, true, true, 'ON', 30, 0, '通用保存成功提示', now(), now()),
    (6, 7, 'prompt.common.delete_confirm', 'PROMPT_COMMON_DELETE_CONFIRM', 'HINT', '确认删除当前数据？', '{"tone":"warning"}'::jsonb, true, true, 'ON', 31, 0, '通用删除确认提示', now(), now()),
    (7, 9, 'device.type.temp_sensor', 'DEVICE_TYPE_TEMP_SENSOR', 'ENUM', '温湿度传感器', '{"deviceType":"sensor","code":"TEMP_SENSOR"}'::jsonb, true, true, 'ON', 40, 0, '设备类型-温湿度传感器', now(), now()),
    (8, 9, 'device.type.current_meter', 'DEVICE_TYPE_CURRENT_METER', 'ENUM', '电流表', '{"deviceType":"meter","code":"CURRENT_METER"}'::jsonb, true, true, 'ON', 41, 0, '设备类型-电流表', now(), now()),
    (9, 9, 'device.type.gas_detector', 'DEVICE_TYPE_GAS_DETECTOR', 'ENUM', '气体探测器', '{"deviceType":"detector","code":"GAS_DETECTOR"}'::jsonb, true, true, 'ON', 42, 0, '设备类型-气体探测器', now(), now());
SELECT setval('sys_dict_labels_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_dict_labels));

INSERT INTO public.sys_dict_label_i18n (
    label_id, language_code, text_value, short_text, description, tenant_id, created_at, updated_at
)
VALUES
    (1, 'zh-CN', '用户列表', '列表', '用户列表页标题', 0, now(), now()),
    (2, 'zh-CN', '用户详情', '详情', '用户详情页标题', 0, now(), now()),
    (3, 'zh-CN', '用户管理', '用户', '系统菜单-用户管理', 0, now(), now()),
    (4, 'zh-CN', '角色管理', '角色', '系统菜单-角色管理', 0, now(), now()),
    (5, 'zh-CN', '保存成功', '成功', '通用保存成功提示', 0, now(), now()),
    (6, 'zh-CN', '确认删除当前数据？', '确认删除', '通用删除确认提示', 0, now(), now()),
    (7, 'zh-CN', '温湿度传感器', '温湿度', '设备类型-温湿度传感器', 0, now(), now()),
    (8, 'zh-CN', '电流表', '电流表', '设备类型-电流表', 0, now(), now()),
    (9, 'zh-CN', '气体探测器', '气体探测', '设备类型-气体探测器', 0, now(), now()),
    (1, 'en-US', 'User List', 'List', 'User list page title', 0, now(), now()),
    (2, 'en-US', 'User Detail', 'Detail', 'User detail page title', 0, now(), now()),
    (3, 'en-US', 'User Management', 'Users', 'System menu - user management', 0, now(), now()),
    (4, 'en-US', 'Role Management', 'Roles', 'System menu - role management', 0, now(), now()),
    (5, 'en-US', 'Saved successfully', 'Saved', 'Common save success prompt', 0, now(), now()),
    (6, 'en-US', 'Are you sure you want to delete this record?', 'Delete?', 'Common delete confirmation prompt', 0, now(), now()),
    (7, 'en-US', 'Temperature Sensor', 'Temp', 'Device type - temperature sensor', 0, now(), now()),
    (8, 'en-US', 'Current Meter', 'Meter', 'Device type - current meter', 0, now(), now()),
    (9, 'en-US', 'Gas Detector', 'Gas', 'Device type - gas detector', 0, now(), now());
SELECT setval('sys_dict_label_i18n_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.sys_dict_label_i18n));

INSERT INTO public.internal_message_categories (id, code, name, remark, sort_order, is_enabled, created_at)
VALUES
    (1, 'order', '订单通知', '包含订单支付、发货、退款等全流程通知', 1, true, now()),
    (101, 'order_paid', '支付成功', '订单支付完成时触发的通知', 2, true, now()),
    (102, 'order_unpaid', '支付超时', '订单未在规定时间内支付的提醒', 3, true, now()),
    (103, 'order_shipped', '已发货', '商家发货后通知用户', 4, true, now()),
    (104, 'order_refunded', '已退款', '订单退款流程完成的通知', 5, true, now()),
    (2, 'system', '系统通知', '系统公告、维护提醒、版本更新等平台级通知', 6, true, now()),
    (201, 'system_announcement', '系统公告', '平台规则更新、重要通知等', 7, true, now()),
    (202, 'system_maintenance', '维护通知', '系统计划维护的时间提醒', 8, true, now()),
    (203, 'system_upgrade', '版本更新', '客户端或功能升级提示', 9, true, now()),
    (3, 'activity', '活动通知', '营销活动报名、开始、结束等提醒', 10, true, now()),
    (301, 'activity_signup', '报名成功', '用户报名活动后的确认通知', 11, true, now()),
    (302, 'activity_start', '活动开始', '活动即将开始的倒计时提醒', 12, true, now()),
    (303, 'activity_end', '活动结束', '活动结束及结果公示通知', 13, true, now()),
    (4, 'user', '用户通知', '账号安全、资料变更、权限调整等通知', 14, true, now()),
    (401, 'user_login_abnormal', '异地登录', '账号在陌生设备登录的安全提醒', 15, true, now()),
    (402, 'user_profile_updated', '资料变更', '用户手机号、邮箱等信息修改后的通知', 16, true, now()),
    (403, 'user_permission_changed', '权限变更', '账号角色或功能权限调整通知', 17, true, now());
SELECT setval('internal_message_categories_id_seq', (SELECT COALESCE(MAX(id), 1) FROM public.internal_message_categories));

COMMIT;
