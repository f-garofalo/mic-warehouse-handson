using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

var issuer = Environment.GetEnvironmentVariable("IAM_ISSUER") ?? "http://iam-mock:9001";
var signingKeyId = Environment.GetEnvironmentVariable("IAM_KEY_ID") ?? "tsid-mock-rs512-2026";
var userClientId = Environment.GetEnvironmentVariable("IAM_USER_CLIENT_ID") ?? "11111111-2222-4333-8444-555555555555";
using var rsa = RSA.Create(2048);

var users = new Dictionary<string, DemoUser>(StringComparer.OrdinalIgnoreCase)
{
    ["alice"] = new(
        Subject: "049823c2-27f9-4507-a5c7-432c3efef275",
        Name: "Alice Demo",
        GivenName: "Alice",
        FamilyName: "Demo",
        Email: "alice@example.com",
        NcsId: "824dd6fd-d439-5e29-b8e6-1a9460664917",
        Locale: "it-IT"),
    ["bob"] = new(
        Subject: "37dd6fad-7a82-4662-85bb-9fd7f9db83b1",
        Name: "Bob Reader",
        GivenName: "Bob",
        FamilyName: "Reader",
        Email: "bob@example.com",
        NcsId: "2a41c9db-1a6a-4d09-9c2e-f83e5ce05a41",
        Locale: "it-IT")
};

var clients = new Dictionary<string, DemoClient>(StringComparer.OrdinalIgnoreCase)
{
    ["svc-orders"] = new("svc-orders", "demo", "Orders Service"),
    [userClientId] = new(userClientId, "demo", "TS AuthorizationCode Client")
};

app.MapGet("/health", () => Results.Ok(new { status = "ok", issuer }));

app.MapGet("/.well-known/openid-configuration", () => Results.Ok(new
{
    issuer,
    token_endpoint = $"{issuer}/oauth/token",
    jwks_uri = $"{issuer}/.well-known/jwks.json",
    response_types_supported = new[] { "code" },
    grant_types_supported = new[] { "authorization_code", "password", "client_credentials", "refresh_token" },
    token_endpoint_auth_methods_supported = new[] { "client_secret_post" },
    id_token_signing_alg_values_supported = new[] { "RS512" }
}));

app.MapGet("/.well-known/jwks.json", () =>
{
    var p = rsa.ExportParameters(false);
    return Results.Ok(new
    {
        keys = new[]
        {
            new
            {
                kty = "RSA",
                use = "sig",
                kid = signingKeyId,
                alg = "RS512",
                n = Base64Url(p.Modulus!),
                e = Base64Url(p.Exponent!)
            }
        }
    });
});

app.MapPost("/oauth/token", async (HttpRequest request) =>
{
    var form = await request.ReadFormAsync();
    var grantType = form["grant_type"].ToString();
    var scope = NormalizeScope(form["scope"].ToString());

    return grantType switch
    {
        "password" => IssueUserToken(form, scope),
        "authorization_code" => IssueUserToken(form, scope),
        "client_credentials" => IssueClientCredentialsToken(form, scope),
        _ => Results.BadRequest(new { error = "unsupported_grant_type" })
    };
});

IResult IssueUserToken(IFormCollection form, string scope)
{
    var username = form["username"].ToString();
    if (string.IsNullOrWhiteSpace(username) && !string.IsNullOrWhiteSpace(form["code"].ToString()))
    {
        username = "alice";
    }
    if (!users.TryGetValue(username, out var user))
    {
        return Results.BadRequest(new { error = "invalid_grant" });
    }

    var clientId = form["client_id"].ToString();
    if (string.IsNullOrWhiteSpace(clientId)) clientId = userClientId;

    var now = DateTimeOffset.UtcNow;
    var sid = UpperHex(16);
    var claims = new Dictionary<string, object>
    {
        ["iss"] = issuer,
        ["nbf"] = now.ToUnixTimeSeconds(),
        ["iat"] = now.ToUnixTimeSeconds(),
        ["exp"] = now.AddHours(1).ToUnixTimeSeconds(),
        ["scope"] = scope.Split(' ', StringSplitOptions.RemoveEmptyEntries),
        ["amr"] = new[] { "external" },
        ["client_id"] = clientId,
        ["sub"] = user.Subject,
        ["auth_time"] = now.AddSeconds(-3).ToUnixTimeSeconds(),
        ["idp"] = "teamsystem",
        ["name"] = user.Name,
        ["given_name"] = user.GivenName,
        ["family_name"] = user.FamilyName,
        ["email"] = user.Email,
        ["ncs_id"] = user.NcsId,
        ["sid"] = sid,
        ["jti"] = UpperHex(16),
        ["locale"] = user.Locale
    };

    var accessToken = SignJwt(claims);
    return Results.Ok(new
    {
        access_token = accessToken,
        refresh_token = UpperHex(32),
        expires_in = 3600,
        token_type = "Bearer",
        scope,
        session_token = Guid.NewGuid().ToString(),
        name = user.Name,
        given_name = user.GivenName,
        family_name = user.FamilyName,
        email = user.Email,
        ncs_id = user.NcsId
    });
}

IResult IssueClientCredentialsToken(IFormCollection form, string scope)
{
    var clientId = form["client_id"].ToString();
    var secret = form["client_secret"].ToString();
    if (!clients.TryGetValue(clientId, out var client) || client.Secret != secret)
    {
        return Results.BadRequest(new { error = "invalid_client" });
    }

    var now = DateTimeOffset.UtcNow;
    var claims = new Dictionary<string, object>
    {
        ["iss"] = issuer,
        ["nbf"] = now.ToUnixTimeSeconds(),
        ["iat"] = now.ToUnixTimeSeconds(),
        ["exp"] = now.AddHours(1).ToUnixTimeSeconds(),
        ["scope"] = scope.Split(' ', StringSplitOptions.RemoveEmptyEntries),
        ["amr"] = new[] { "client_credentials" },
        ["client_id"] = client.ClientId,
        ["sub"] = client.ClientId,
        ["idp"] = "teamsystem",
        ["name"] = client.DisplayName,
        ["sid"] = UpperHex(16),
        ["jti"] = UpperHex(16)
    };

    return Results.Ok(new
    {
        access_token = SignJwt(claims),
        expires_in = 3600,
        token_type = "Bearer",
        scope
    });
}

string SignJwt(Dictionary<string, object> claims)
{
    var header = new Dictionary<string, object>
    {
        ["alg"] = "RS512",
        ["kid"] = signingKeyId,
        ["typ"] = "at+jwt"
    };
    var headerPart = Base64Url(JsonSerializer.SerializeToUtf8Bytes(header));
    var payloadPart = Base64Url(JsonSerializer.SerializeToUtf8Bytes(claims));
    var signingInput = Encoding.ASCII.GetBytes($"{headerPart}.{payloadPart}");
    var signature = rsa.SignData(signingInput, HashAlgorithmName.SHA512, RSASignaturePadding.Pkcs1);
    return $"{headerPart}.{payloadPart}.{Base64Url(signature)}";
}

static string NormalizeScope(string? scope)
{
    if (string.IsNullOrWhiteSpace(scope)) return "openid profile offline_access";
    return string.Join(' ', scope.Split(new[] { ' ', ',', ';' }, StringSplitOptions.RemoveEmptyEntries));
}

static string Base64Url(byte[] bytes) => Convert.ToBase64String(bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_');

static string UpperHex(int bytes)
{
    var data = RandomNumberGenerator.GetBytes(bytes);
    return Convert.ToHexString(data);
}

app.Run();

record DemoUser(string Subject, string Name, string GivenName, string FamilyName, string Email, string NcsId, string Locale);
record DemoClient(string ClientId, string Secret, string DisplayName);
