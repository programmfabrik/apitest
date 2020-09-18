DROP TABLE IF EXISTS "test_endpoints";

CREATE TABLE "test_endpoints" (
    "endpoint" TEXT NOT NULL,
    "method" TEXT NOT NULL DEFAULT 'GET',
    "file" TEXT,
    "format" BLOB,
    "param1"  TEXT,
    "param2"  INTEGER,
    "param3"  NUMERIC,
    "expected_statuscode" INTEGER NOT NULL DEFAULT 200,
    "expected_body" BLOB
);

INSERT INTO "test_endpoints" ("endpoint", "method", "param1", "param2", "param3", "expected_body")
VALUES ('bounce-json', 'POST', 'abc', 123, true, '{"query_params": {"param1": ["abc"],"param2": ["123"],"param3": ["1"]}}');

INSERT INTO "test_endpoints" ("endpoint", "method", "param1", "expected_body")
VALUES ('bounce-json', 'POST', 'xyz', '{"query_params": {"param1": ["xyz"]}}');

INSERT INTO "test_endpoints" ("endpoint", "method", "param1", "param2", "param3", "file", "format", "expected_body")
VALUES ('bounce', 'POST', 'abc', 123, true, '../_res/assets/camera.jpg', '{"pre_process": {"cmd": {"name": "exiftool","args": ["-j", "-g", "-"]}}}', '[{"SourceFile": "-","ExifTool": {},"JFIF": {},"ICC_Profile": {},"File": {},"Composite": {}}]');

INSERT INTO "test_endpoints" ("endpoint", "expected_statuscode")
VALUES ('wrong/path.jpg', 404);

INSERT INTO "test_endpoints" ("endpoint", "format")
VALUES ('assets/camera.jpg', '{"pre_process": {"cmd": {"name": "exiftool","args": ["-j", "-g", "-"]}}}');
