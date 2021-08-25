-- Create a new database with the name 'fullstack_otp'.
DROP DATABASE IF EXISTS fullstack_otp;
CREATE DATABASE IF NOT EXISTS fullstack_otp;
USE fullstack_otp;

-- Create a table for authentication.
CREATE TABLE users (
  id          VARCHAR(255) NOT NULL,
  username    VARCHAR(255) NOT NULL UNIQUE,
  email       VARCHAR(255) NOT NULL UNIQUE,
  phone       VARCHAR(255) NOT NULL UNIQUE,
  role        ENUM ('admin', 'user') NOT NULL,
  password    VARCHAR(255) NOT NULL,
  created     VARCHAR(255) NOT NULL,
  modified    VARCHAR(255) NOT NULL,

  PRIMARY KEY (id)
) ENGINE=InnoDB CHARACTER SET utf8;

-- Create a table to store senstive data.
CREATE TABLE data (
  id                VARCHAR(255) NOT NULL,
  citizen_id        TEXT NOT NULL UNIQUE,
  ssn               TEXT NOT NULL,
  name              TEXT NOT NULL,
  address           TEXT NOT NULL,
  gender            ENUM ('male', 'female', 'unspecified', 'others') NOT NULL,
  race              TEXT NOT NULL,
  religion          TEXT NOT NULL,
  bank_account      TEXT NOT NULL,
  bank_name         TEXT NOT NULL,
  credit_card       TEXT,
  aid               INT,
  fines             INT,
  drugs             TEXT,
  notes             TEXT,
  created           VARCHAR(255) NOT NULL,
  modified          VARCHAR(255) NOT NULL,

  PRIMARY KEY (id)
) ENGINE=InnoDB CHARACTER SET utf8;

-- Populate with sample data.
-- For the superuser, you can log in with 'kaede' and 'admin'. Development only.
INSERT INTO users VALUES
("a5d7cba6-1f26-45b9-8133-b60daa3e6b73", "kaede", "kaede@mail.co.jp", "+628381223344", "admin", "$2a$14$vALUjLRYLScIl19PIdXoz.7RzzbAzJjN3lTcsMvyvm3WP77fzms0W", UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
("8aa25f92-1721-4747-b9a6-72161fcd4ccb", "sayu", "sayu@mail.co.jp", "+6281233445566", "user", "$2a$14$vjXi/pR9xSMwF9GoVW/L4uP2LasVS6G7jl3YLwULaYF.YmYGAqrHq", UNIX_TIMESTAMP(), UNIX_TIMESTAMP());

-- Populate with sample sensitive data.
INSERT INTO data VALUES
("e8b3ece3-5375-47af-b2c1-763921057530", "21849011223344", "312800989500", "Adi Nugroho", "Alam Segar IX, Pondok Indah, South Jakarta", "male", "Javanese", "Islam", "6010234802", "Bank Central Asia", "1254024511220000", 500000, 25000, "Opium", "Friendly and outgoing, but sometimes too arrogant.", UNIX_TIMESTAMP(), UNIX_TIMESTAMP());
