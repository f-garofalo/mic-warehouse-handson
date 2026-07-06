<?php
declare(strict_types=1);

namespace MIC;

/**
 * Tiny request router. Pattern syntax: /api/foo/:id/bar
 * The :name segments are passed to the handler as $params['name'].
 */
final class Router
{
    /** @var array<int, array{method:string, pattern:string, regex:string, handler:callable}> */
    private array $routes = [];

    public function add(string $method, string $pattern, callable $handler): void
    {
        $regex = preg_replace('#:([a-zA-Z_][a-zA-Z0-9_]*)#', '(?P<$1>[^/]+)', $pattern);
        $regex = '#^' . $regex . '/?$#';
        $this->routes[] = [
            'method'  => strtoupper($method),
            'pattern' => $pattern,
            'regex'   => $regex,
            'handler' => $handler,
        ];
    }

    public function get(string $p, callable $h): void    { $this->add('GET',    $p, $h); }
    public function post(string $p, callable $h): void   { $this->add('POST',   $p, $h); }
    public function put(string $p, callable $h): void    { $this->add('PUT',    $p, $h); }
    public function delete(string $p, callable $h): void { $this->add('DELETE', $p, $h); }

    /**
     * Dispatch and return [status, body] OR null when no route matched.
     * @return array{0:int, 1:mixed}|null
     */
    public function dispatch(string $method, string $path): ?array
    {
        $method = strtoupper($method);
        foreach ($this->routes as $r) {
            if ($r['method'] !== $method) continue;
            if (preg_match($r['regex'], $path, $m)) {
                $params = [];
                foreach ($m as $k => $v) {
                    if (is_string($k)) $params[$k] = $v;
                }
                $body = ($r['handler'])($params);
                if (is_array($body) && isset($body[0]) && is_int($body[0]) && count($body) === 2) {
                    return $body; // handler returned [status, payload]
                }
                return [200, $body];
            }
        }
        return null;
    }
}
