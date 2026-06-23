def test_read_file():
    data = read_file("builtins_test.star")
    assert.true("def test_read_file():" in data)

def test_fail_read_file_traversal():
    read_file("../data.txt")

def test_fail_read_file_absolute():
    read_file("/etc/passwd")

def test_txtar():
    archive = txtar("""-- file1.txt --
hello
-- file2.txt --
world
""")
    assert.eq(len(archive), 2)
    assert.eq(archive["file1.txt"], "hello\n")
    assert.eq(archive["file2.txt"], "world\n")

def test_fail_txtar_invalid_arg():
    txtar(123)
