from pathlib import Path
import yaml

ROOT = Path("/contract") if Path("/contract/api").exists() else Path(__file__).resolve().parents[2]


def ref(operation, schema):
    content = operation.get("responses", {})
    for response in content.values():
        for media in response.get("content", {}).values():
            if media.get("schema", {}).get("$ref") == f"#/components/schemas/{schema}":
                return True
    return False


def assert_security(operation, expected):
    assert operation.get("security") == expected, operation


def assert_problem(operation, statuses):
    for status in statuses:
        response = operation["responses"][str(status)]
        assert "application/problem+json" in response.get("content", {}), (status, response)
        assert ref({"responses": {str(status): response}}, "ProblemDetails"), (status, response)


def main():
    bank = yaml.safe_load((ROOT / "api/bank.openapi.yaml").read_text())
    network = yaml.safe_load((ROOT / "api/network.openapi.yaml").read_text())
    assert bank["openapi"] == "3.1.0"
    assert network["openapi"] == "3.1.0"

    # T3 contract remains present.
    for path, method in [
        ("/v1/participants/{participantId}", "get"),
        ("/v1/payment-intents/{paymentIntentId}", "get"),
        ("/v1/payment-intents/{paymentIntentId}/refunds", "post"),
        ("/v1/accounts/{accountId}/movements", "get"),
        ("/v1/accounts/{accountId}/journals", "get"),
    ]:
        assert method in bank["paths"][path]

    public = {
        "/v1/keys": ("post", "CholoKeyCreate", "CholoKey"),
        "/v1/keys/{keyId}": ("get", None, "CholoKey"),
        "/v1/keys/{keyId}/suspend": ("post", None, None),
        "/v1/keys/{keyId}/block": ("post", None, None),
        "/v1/keys/{keyId}/delete": ("post", None, None),
        "/v1/keys/{keyId}/qr": ("get", None, "QrResponse"),
        "/v1/keys/{keyId}/qr/dynamic": ("post", "DynamicQrCreate", "QrResponse"),
        "/v1/qr/verify": ("post", "QrVerify", "QrVerification"),
    }
    for path, (method, request_schema, response_schema) in public.items():
        assert path in bank["paths"]
        operation = bank["paths"][path][method]
        expected_security = [] if path == "/v1/qr/verify" else [{"bearerAuth": []}]
        assert_security(operation, expected_security)
        if request_schema:
            body = operation["requestBody"]
            assert body["required"] is True
            assert body["content"]["application/json"]["schema"]["$ref"].endswith(request_schema)
        if response_schema:
            assert ref(operation, response_schema)
        assert "responses" in operation

    assert "get" not in bank["paths"]["/v1/keys"]  # no public enumeration
    assert_problem(bank["paths"]["/v1/keys"]["post"], [400, 401, 409, 503])
    assert_problem(bank["paths"]["/v1/keys/{keyId}"]["get"], [404, 503])
    for path in ["/v1/keys/{keyId}/suspend", "/v1/keys/{keyId}/block", "/v1/keys/{keyId}/delete"]:
        assert_problem(bank["paths"][path]["post"], [400, 404, 409])
    assert_problem(bank["paths"]["/v1/keys/{keyId}/qr"]["get"], [404, 409])
    assert_problem(bank["paths"]["/v1/keys/{keyId}/qr/dynamic"]["post"], [400, 404])
    assert_problem(bank["paths"]["/v1/qr/verify"]["post"], [409])

    internal = {
        "/internal/v1/directory/keys": "post",
        "/internal/v1/directory/keys/{keyID}": "get",
        "/internal/v1/directory/keys/{keyID}/suspend": "post",
        "/internal/v1/directory/keys/{keyID}/block": "post",
        "/internal/v1/directory/keys/{keyID}/delete": "post",
        "/internal/v1/directory/resolve": "post",
        "/internal/v1/directory/port": "post",
    }
    for path, method in internal.items():
        assert path in network["paths"] and method in network["paths"][path], (path, method)
        assert network["paths"][path][method].get("security") == [{"serviceBearer": []}]
    resolve = network["paths"]["/internal/v1/directory/resolve"]["post"]
    assert resolve["requestBody"]["content"]["application/json"]["schema"]["$ref"].endswith("DirectoryResolve")
    assert ref(resolve, "DirectoryResolution")
    create = network["paths"]["/internal/v1/directory/keys"]["post"]
    assert create["requestBody"]["content"]["application/json"]["schema"]["$ref"].endswith("DirectoryKeyCreate")
    assert ref(create, "DirectoryKey")
    for suffix in ("suspend", "block", "delete"):
        operation = network["paths"][f"/internal/v1/directory/keys/{{keyID}}/{suffix}"]["post"]
        assert operation["requestBody"]["content"]["application/json"]["schema"]["$ref"].endswith("DirectoryLifecycle")
        assert "202" in operation["responses"]
    assert "/v1/keys/{keyId}" not in network["paths"]

    for schema in ["ProblemDetails", "CholoKey", "CholoKeyCreate", "DynamicQrCreate", "QrResponse", "QrVerify", "QrVerification"]:
        assert schema in bank["components"]["schemas"]
    for schema in ["DirectoryKey", "DirectoryKeyCreate", "DirectoryLifecycle", "DirectoryResolve", "DirectoryResolution"]:
        assert schema in network["components"]["schemas"]
    print("OpenAPI T4c contract validation passed")


if __name__ == "__main__":
    main()
