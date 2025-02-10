ALTER TABLE clusterephemeralreports DROP COLUMN cluster_id UUID;
ALTER TABLE ephemeralreports DROP COLUMN cluster_id UUID;
ALTER TABLE clusterpolicyreports DROP COLUMN cluster_id UUID;
ALTER TABLE policyreports DROP COLUMN cluster_id UUID;