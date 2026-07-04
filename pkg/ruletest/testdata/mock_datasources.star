def test_mock_datasource_passing():
    res = eval(
        rule="rule_type_mock_datasource.yaml",
        entity={"type": "repository", "name": "test"},
        profile={"required_value": "hello"},
        data_sources={
            "mock_ds.get_val": "hello"
        }
    )
    assert.eq(res["status"], "pass")

def test_mock_datasource_failing():
    res = eval(
        rule="rule_type_mock_datasource.yaml",
        entity={"type": "repository", "name": "test"},
        profile={"required_value": "hello"},
        data_sources={
            "mock_ds.get_val": "wrong_value"
        }
    )
    assert.eq(res["status"], "fail")
