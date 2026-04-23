// C# fixture for the tree-sitter indexer.
//
// Exercises: using directives, namespace, interface, class + methods,
// struct, record, and a static top-level method.

using System;
using System.Security.Cryptography;
using Microsoft.Extensions.Logging;

namespace App.Auth
{
    public interface ITokenVerifier
    {
        bool Verify(string token, string secret);
    }

    public struct Credentials
    {
        public string Username;
        public string Secret;
    }

    public record TokenPayload(string Subject, DateTime IssuedAt);

    public class HmacTokenVerifier : ITokenVerifier
    {
        private readonly string _realm;
        private readonly ILogger<HmacTokenVerifier> _logger;

        public HmacTokenVerifier(string realm, ILogger<HmacTokenVerifier> logger)
        {
            _realm = realm;
            _logger = logger;
        }

        public bool Verify(string token, string secret)
        {
            var expected = Issue(secret, _realm);
            return string.Equals(expected, token, StringComparison.Ordinal);
        }

        public string Issue(string subject, string secret)
        {
            using var hmac = new HMACSHA256(Convert.FromBase64String(secret));
            var bytes = hmac.ComputeHash(System.Text.Encoding.UTF8.GetBytes($"{_realm}:{subject}"));
            return Convert.ToBase64String(bytes);
        }
    }

    public static class AuthFactory
    {
        public static HmacTokenVerifier DefaultVerifier(ILogger<HmacTokenVerifier> logger)
        {
            return new HmacTokenVerifier("app", logger);
        }
    }
}
