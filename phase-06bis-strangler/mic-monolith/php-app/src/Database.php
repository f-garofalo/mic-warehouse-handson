<?php
declare(strict_types=1);

namespace MIC;

use PDO;
use PDOException;

/**
 * Singleton-ish PDO wrapper. Retries on cold start (waits for MySQL).
 */
final class Database
{
    private static ?PDO $pdo = null;

    public static function get(): PDO
    {
        if (self::$pdo !== null) {
            return self::$pdo;
        }

        $host = getenv('DB_HOST') ?: 'mysql';
        $port = getenv('DB_PORT') ?: '3306';
        $name = getenv('DB_NAME') ?: 'mic';
        $user = getenv('DB_USER') ?: 'root';
        $pass = getenv('DB_PASS') ?: 'root';

        $dsn = "mysql:host={$host};port={$port};dbname={$name};charset=utf8mb4";

        $attempts = 0;
        $maxAttempts = 30;
        while (true) {
            try {
                self::$pdo = new PDO($dsn, $user, $pass, [
                    PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION,
                    PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
                    PDO::ATTR_EMULATE_PREPARES   => false,
                ]);
                return self::$pdo;
            } catch (PDOException $e) {
                $attempts++;
                if ($attempts >= $maxAttempts) {
                    throw $e;
                }
                usleep(500_000); // 0.5s
            }
        }
    }
}
