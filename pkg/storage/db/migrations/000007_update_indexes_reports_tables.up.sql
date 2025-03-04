ALTER TABLE clusterephemeralreports DROP CONSTRAINT clusterephemeralreports_pkey;
ALTER TABLE clusterephemeralreports ADD PRIMARY KEY (cluster_id, name);
ALTER TABLE clusterpolicyreports DROP CONSTRAINT clusterpolicyreports_pkey;
ALTER TABLE clusterpolicyreports ADD PRIMARY KEY (cluster_id, name);
ALTER TABLE ephemeralreports DROP CONSTRAINT ephemeralreports_pkey;
ALTER TABLE ephemeralreports ADD PRIMARY KEY (cluster_id, name, namespace);
CREATE INDEX IF NOT EXISTS ephemeralreports_namespace ON ephemeralreports(cluster_id, namespace);
DROP INDEX IF EXISTS ephemeralreportnamespace;
ALTER TABLE policyreports DROP CONSTRAINT policyreports_pkey;
ALTER TABLE policyreports ADD PRIMARY KEY (cluster_id, name, namespace);
CREATE INDEX IF NOT EXISTS policyreports_namespace ON policyreports(cluster_id, namespace);
DROP INDEX IF EXISTS policyreportnamespace;
