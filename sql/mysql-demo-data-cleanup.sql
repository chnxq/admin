-- Description: cleanup MySQL demo seed data only
-- Rule: preserve Go bootstrap base data, delete only demo data owned by mysql-demo-data.sql

SET FOREIGN_KEY_CHECKS = 0;
START TRANSACTION;

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

COMMIT;
SET FOREIGN_KEY_CHECKS = 1;

SELECT 'MySQL demo seed cleanup completed' AS result;
