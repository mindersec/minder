def test_mock_datasource_passing():
    res = eval(
        rule="mock_datasource_rule",
        entity={"type": "repository", "name": "test"},
        profile={"required_value": "hello"},
        data_sources=["testdata/mock_datasource_def.yaml"],
        mock_http={
            "https://api.github.com/mock_endpoint": body('"hello"')
        }
    )
    if res["status"] != "pass":
        assert.fail("expected pass, got %s: %s" % (res["status"], res["message"]))
    assert.eq(res["status"], "pass")

def test_mock_datasource_failing():
    res = eval(
        rule="mock_datasource_rule",
        entity={"type": "repository", "name": "test"},
        profile={"required_value": "hello"},
        data_sources=["testdata/mock_datasource_def.yaml"],
        mock_http={
            "https://api.github.com/mock_endpoint": body('"wrong_value"')
        }
    )
    if res["status"] != "fail":
        print(res["message"])
    assert.eq(res["status"], "fail")
