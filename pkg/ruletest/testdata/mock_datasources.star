def test_mock_datasource_passing():
    res = eval(
        rule="mock_datasource_rule",
        entity={"type": "repository", "name": "test"},
        profile={"required_value": "hello"},
        data_sources={
            "mock_ds.get_val": "hello"
        }
    )
    if res["status"] != "pass":
        print(res["message"])
    assert.eq(res["status"], "pass")

def test_mock_datasource_failing():
    res = eval(
        rule="mock_datasource_rule",
        entity={"type": "repository", "name": "test"},
        profile={"required_value": "hello"},
        data_sources={
            "mock_ds.get_val": "wrong_value"
        }
    )
    if res["status"] != "fail":
        print(res["message"])
    assert.eq(res["status"], "fail")
