ALTER TABLE clusterreports DROP CONSTRAINT clusterreports_pkey;
ALTER TABLE clusterreports ADD PRIMARY KEY (name);
ALTER TABLE reports DROP CONSTRAINT reports_pkey;
ALTER TABLE reports ADD PRIMARY KEY (name, namespace);
CREATE INDEX IF NOT EXISTS reportnamespace ON reports(namespace);
DROP INDEX IF EXISTS reports_namespace;
