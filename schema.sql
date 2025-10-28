CREATE TABLE domains (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL UNIQUE,
  master VARCHAR(128) DEFAULT NULL,
  last_check INT DEFAULT NULL,
  type VARCHAR(6) NOT NULL,
  notified_serial INT DEFAULT NULL,
  options VARCHAR(65535) DEFAULT NULL,
  account VARCHAR(40) DEFAULT NULL,
  catalog VARCHAR(255) DEFAULT NULL
);

CREATE INDEX name_index ON domains(name);

CREATE TABLE records (
  id SERIAL PRIMARY KEY,
  domain_id INT DEFAULT NULL,
  name VARCHAR(255) DEFAULT NULL,
  type VARCHAR(10) DEFAULT NULL,
  content VARCHAR(65535) DEFAULT NULL,
  ttl INT DEFAULT NULL,
  prio INT DEFAULT NULL,
  disabled BOOL DEFAULT false,
  ordername VARCHAR(255),
  auth BOOL DEFAULT true,
  CONSTRAINT domain_id_fk FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
);

CREATE INDEX rec_name_index ON records(name);
CREATE INDEX nametype_index ON records(name, type);
CREATE INDEX domain_id_index ON records(domain_id);

CREATE TABLE supermasters (
  ip INET NOT NULL,
  nameserver VARCHAR(255) NOT NULL,
  account VARCHAR(40) DEFAULT NULL,
  PRIMARY KEY (ip, nameserver)
);

CREATE TABLE domainmetadata (
  id SERIAL PRIMARY KEY,
  domain_id INT REFERENCES domains(id) ON DELETE CASCADE,
  kind VARCHAR(32),
  content TEXT
);

CREATE INDEX domainmetadata_idx ON domainmetadata(domain_id);

CREATE TABLE comments (
  id SERIAL PRIMARY KEY,
  domain_id INT REFERENCES domains(id) ON DELETE CASCADE,
  name VARCHAR(255),
  type VARCHAR(255),
  modified_at INT,
  account VARCHAR(40),
  comment TEXT
);

CREATE INDEX comments_idx ON comments(domain_id);

CREATE TABLE cryptokeys (
  id SERIAL PRIMARY KEY,
  domain_id INT REFERENCES domains(id) ON DELETE CASCADE,
  flags INT NOT NULL,
  active BOOL,
  published BOOL,
  content TEXT
);

CREATE INDEX cryptokeys_idx ON cryptokeys(domain_id);

CREATE TABLE tsigkeys (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) UNIQUE,
  algorithm VARCHAR(50),
  secret VARCHAR(255)
);