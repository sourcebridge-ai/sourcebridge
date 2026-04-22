<?php

/**
 * Minimal authentication helper for the indexer fixture.
 *
 * Covers: namespace use, class + interface + method extraction,
 * function call nodes, and PHPDoc comment capture.
 */

namespace App\Auth;

use Psr\Log\LoggerInterface;
use App\Session\Store;

interface TokenVerifier
{
    public function verify(string $token, string $secret): bool;
}

class HmacTokenVerifier implements TokenVerifier
{
    private string $realm;
    private LoggerInterface $logger;

    public function __construct(string $realm, LoggerInterface $logger)
    {
        $this->realm = $realm;
        $this->logger = $logger;
    }

    public function verify(string $token, string $secret): bool
    {
        $expected = hash_hmac('sha256', $this->realm . ':' . $secret, $token);
        return hash_equals($expected, $token);
    }

    public function issue(string $subject, string $secret): string
    {
        return hash_hmac('sha256', $this->realm . ':' . $subject, $secret);
    }
}

function default_verifier(LoggerInterface $logger): HmacTokenVerifier
{
    return new HmacTokenVerifier('app', $logger);
}
