ALTER TABLE clusterreports DROP CONSTRAINT clusterreports_pkey;
ALTER TABLE clusterreports ADD PRIMARY KEY (cluster_id, name);
ALTER TABLE reports DROP CONSTRAINT reports_pkey;
ALTER TABLE reports ADD PRIMARY KEY (cluster_id, name, namespace);
CREATE INDEX IF NOT EXISTS reports_namespace ON reports(cluster_id, namespace);
DROP INDEX IF EXISTS reportnamespace;