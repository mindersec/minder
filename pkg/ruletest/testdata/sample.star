# Simple test that passes
def test_passing():
    pass

# Simple test that fails using the built-in fail()
def test_failing():
    fail("this test failed intentionally")

# Test that throws a Starlark exception
def test_exception():
    1 / 0

# A helper function, not a test (does not start with test_)
def helper():
    pass

