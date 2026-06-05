BEGIN;

SET LOCAL search_path = public, pg_catalog;

DELETE FROM public.sys_task_logs WHERE id BETWEEN 1701 AND 1799;
DELETE FROM public.sys_tasks WHERE id BETWEEN 1601 AND 1699;
DELETE FROM public.sys_task_groups WHERE id BETWEEN 1501 AND 1599;
DELETE FROM public.sys_login_policies WHERE id BETWEEN 1801 AND 1899;
DELETE FROM public.sys_dict_label_i18n WHERE label_id BETWEEN 2001 AND 2099;
DELETE FROM public.sys_dict_labels WHERE id BETWEEN 2001 AND 2099;
DELETE FROM public.sys_dict_category_i18n WHERE category_id BETWEEN 1901 AND 1999;
DELETE FROM public.sys_dict_categories WHERE id BETWEEN 1901 AND 1999;
DELETE FROM public.internal_message_categories WHERE id BETWEEN 2101 AND 2199;

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

COMMIT;
