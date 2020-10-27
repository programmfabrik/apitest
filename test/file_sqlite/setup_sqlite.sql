DROP TABLE IF EXISTS "test_values";

CREATE TABLE "test_values" (
    "text" TEXT NOT NULL,
    "number"  INTEGER,
    "bool"  NUMERIC
);

INSERT INTO "test_values" ("text", "number", "bool")
VALUES ('one', 1, true);

INSERT INTO "test_values" ("text", "number", "bool")
VALUES ('two', 2, false);

INSERT INTO "test_values" ("text", "number")
VALUES ('three', 3);

INSERT INTO "test_values" ("text")
VALUES ('four');
