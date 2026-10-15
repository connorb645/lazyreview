import { test } from 'node:test';
import assert from 'node:assert/strict';
import { splitDiff, touches } from '../../src/build/diff/diff-parser.mjs';

const DIFF = `diff --git a/src/Old.swift b/src/Old.swift
index 1111111..2222222 100644
--- a/src/Old.swift
+++ b/src/Old.swift
@@ -1,4 +1,5 @@
 struct A {
-    let a: Int
+    let a: Int64
+    let b: Int
 }

@@ -10,3 +11,2 @@ struct B {
 one
-two
 three
diff --git a/src/New.swift b/src/New.swift
new file mode 100644
index 0000000..3333333
--- /dev/null
+++ b/src/New.swift
@@ -0,0 +1,2 @@
+struct New {}
+
diff --git a/gone.txt b/gone.txt
deleted file mode 100644
index 4444444..0000000
--- a/gone.txt
+++ /dev/null
@@ -1 +0,0 @@
-bye
diff --git a/before name.md b/after name.md
similarity index 100%
rename from before name.md
rename to after name.md
diff --git a/logo.png b/logo.png
index 5555555..6666666 100644
Binary files a/logo.png and b/logo.png differ
`;

test('splits a diff into files with their status and paths', () => {
  const files = splitDiff(DIFF);
  assert.deepEqual(files.map(f => [f.path, f.oldPath, f.status, f.binary]), [
    ['src/Old.swift', 'src/Old.swift', 'M', false],
    ['src/New.swift', null, 'A', false],
    ['gone.txt', 'gone.txt', 'D', false],
    ['after name.md', 'before name.md', 'R', false],
    ['logo.png', 'logo.png', 'M', true],
  ]);
  assert.ok(files[0].text.startsWith('diff --git a/src/Old.swift'));
  assert.ok(files[0].text.endsWith(' three'));
});

test('reads each hunk\'s added lines and removal points on the new side', () => {
  const [old] = splitDiff(DIFF);
  assert.equal(old.hunks.length, 2);
  assert.deepEqual(old.hunks[0].added, [2, 3]);
  assert.deepEqual(old.hunks[0].removedAt, [2]);
  assert.deepEqual(old.hunks[1].added, []);
  assert.deepEqual(old.hunks[1].removedAt, [12]);
});

test('a span is touched by an added line or by lines removed inside it', () => {
  const [old] = splitDiff(DIFF);
  assert.equal(touches(old.hunks, 1, 4), true);
  assert.equal(touches(old.hunks, 11, 13), true);
  assert.equal(touches(old.hunks, 12, 13), false);
  assert.equal(touches(old.hunks, 20, 30), false);
});

test('quoted paths with escaped characters are read back', () => {
  const [file] = splitDiff('diff --git "a/caf\\303\\251.txt" "b/caf\\303\\251.txt"\nindex 1..2 100644\n--- "a/caf\\303\\251.txt"\n+++ "b/caf\\303\\251.txt"\n@@ -1 +1 @@\n-a\n+b\n');
  assert.equal(file.path, 'café.txt');
});
