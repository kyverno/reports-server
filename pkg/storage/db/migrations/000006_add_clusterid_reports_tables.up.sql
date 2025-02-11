ALTER TABLE clusterephemeralreports ADD COLUMN cluster_id UUID;
ALTER TABLE ephemeralreports ADD COLUMN cluster_id UUID;
ALTER TABLE clusterpolicyreports ADD COLUMN cluster_id UUID;
ALTER TABLE policyreports ADD COLUMN cluster_id UUID;