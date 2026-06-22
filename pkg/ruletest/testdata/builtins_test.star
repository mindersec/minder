def test_read_file():
    data = read_file("builtins_test.star")
    assert.true("def test_read_file():" in data)

def test_read_file_traversal():
    assert.fails(lambda: read_file("../data.txt"), "invalid argument")

def test_read_file_absolute():
    assert.fails(lambda: read_file("/etc/passwd"), "invalid argument")

def test_txtar():
    archive = txtar("""-- file1.txt --
hello
-- file2.txt --
world
""")
    assert.eq(len(archive), 2)
    assert.eq(archive["file1.txt"], "hello\n")
    assert.eq(archive["file2.txt"], "world\n")
