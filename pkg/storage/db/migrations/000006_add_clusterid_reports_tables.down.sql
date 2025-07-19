ALTER TABLE clusterephemeralreports DROP COLUMN cluster_id;
ALTER TABLE ephemeralreports DROP COLUMN cluster_id;
ALTER TABLE clusterpolicyreports DROP COLUMN cluster_id;
ALTER TABLE policyreports DROP COLUMN cluster_id;
ALTER TABLE clusterreports DROP COLUMN cluster_id;
ALTER TABLE reports DROP COLUMN cluster_id;