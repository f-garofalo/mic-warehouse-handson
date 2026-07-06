<?php
declare(strict_types=1);

/**
 * MIC - front controller
 *
 * Serves three things:
 *  1) /api/...            -> JSON API (Router + controllers)
 *  2) /                   -> SPA shell (public/index.html)
 *  3) /css/* /js/*        -> static assets from public/
 */

namespace MIC;

use MIC\Controllers\AgenteController;
use MIC\Controllers\ArticleController;
use MIC\Controllers\AuditController;
use MIC\Controllers\CategoryController;
use MIC\Controllers\CustomerController;
use MIC\Controllers\DashboardController;
use MIC\Controllers\InvoiceController;
use MIC\Controllers\IvaController;
use MIC\Controllers\ListinoController;
use MIC\Controllers\MagazzinoController;
use MIC\Controllers\MovimentoController;
use MIC\Controllers\NotaCreditoController;
use MIC\Controllers\OrderController;
use MIC\Controllers\PagamentoController;
use MIC\Controllers\ScontoController;
use MIC\Controllers\SupplierController;
use MIC\Controllers\UserController;

// ---------- PSR-style autoloader (no Composer) ----------
spl_autoload_register(function (string $class): void {
    $prefix = 'MIC\\';
    if (strncmp($class, $prefix, strlen($prefix)) !== 0) return;
    $rel  = substr($class, strlen($prefix));
    $path = __DIR__ . '/src/' . str_replace('\\', '/', $rel) . '.php';
    if (is_file($path)) require_once $path;
});

// ---------- request parsing ----------
$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
$uri    = parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH) ?: '/';

// Static asset (only when nginx didn't already serve it; useful for dev / `php -S`)
if (preg_match('#^/(css|js|favicon\.ico)#', $uri)) {
    $file = __DIR__ . '/public' . $uri;
    if (is_file($file)) {
        $mime = match (pathinfo($file, PATHINFO_EXTENSION)) {
            'css'  => 'text/css',
            'js'   => 'application/javascript',
            'ico'  => 'image/x-icon',
            'png'  => 'image/png',
            'svg'  => 'image/svg+xml',
            default => 'application/octet-stream',
        };
        header("Content-Type: $mime");
        readfile($file);
        return;
    }
    http_response_code(404);
    return;
}

// API requests
if (strncmp($uri, '/api/', 5) === 0) {
    return handleApi($method, $uri);
}

// Anything else: SPA shell
header('Content-Type: text/html; charset=utf-8');
readfile(__DIR__ . '/public/index.html');
return;

// =========================================================================
// API dispatch
// =========================================================================
function handleApi(string $method, string $uri): void
{
    header('Content-Type: application/json; charset=utf-8');
    header('Cache-Control: no-store');

    $r = new Router();

    // ---- Dashboard ----
    $dash = new DashboardController();
    $r->get('/api/dashboard/kpi', fn() => $dash->kpi());

    // ---- helper to register a CRUD set ----
    $crud = function (Router $r, string $base, $ctl) {
        $r->get   ($base,            fn()       => $ctl->index());
        $r->get   ($base . '/:id',   fn($p)     => $ctl->show((int)$p['id']));
        $r->post  ($base,            fn()       => $ctl->create());
        $r->put   ($base . '/:id',   fn($p)     => $ctl->update((int)$p['id']));
        $r->delete($base . '/:id',   fn($p)     => $ctl->destroy((int)$p['id']));
    };

    // anagrafica
    $cust = new CustomerController(); $crud($r, '/api/customers', $cust);
    $r->get('/api/customers/:id/orders',   fn($p) => $cust->orders((int)$p['id']));
    $r->get('/api/customers/:id/invoices', fn($p) => $cust->invoices((int)$p['id']));

    $sup = new SupplierController(); $crud($r, '/api/suppliers', $sup);

    // catalogo
    $art = new ArticleController(); $crud($r, '/api/articles', $art);
    $cat = new CategoryController(); $crud($r, '/api/categories', $cat);
    $iva = new IvaController(); $crud($r, '/api/iva-rates', $iva);
    $list = new ListinoController(); $crud($r, '/api/listini', $list);
    $r->get ('/api/listini/:id/voci', fn($p) => $list->voci((int)$p['id']));
    $r->post('/api/listini/:id/voci', fn($p) => $list->addVoce((int)$p['id']));

    // vendita
    $ord = new OrderController(); $crud($r, '/api/orders', $ord);
    $r->get ('/api/orders/:id/righe', fn($p) => $ord->righe((int)$p['id']));
    $r->post('/api/orders/:id/righe', fn($p) => $ord->addRiga((int)$p['id']));
    $sco = new ScontoController(); $crud($r, '/api/sconti', $sco);
    $age = new AgenteController(); $crud($r, '/api/agenti', $age);

    // logistica
    $mag = new MagazzinoController(); $crud($r, '/api/magazzini', $mag);
    $r->get('/api/magazzini/:id/giacenze', fn($p) => $mag->giacenze((int)$p['id']));
    $mov = new MovimentoController(); $crud($r, '/api/movimenti', $mov);

    // fiscale
    $inv = new InvoiceController(); $crud($r, '/api/invoices', $inv);
    $r->post('/api/invoices/:id/invia-sdi', fn($p) => $inv->inviaSdi((int)$p['id']));
    $nc  = new NotaCreditoController(); $crud($r, '/api/note-credito', $nc);
    $pag = new PagamentoController(); $crud($r, '/api/pagamenti', $pag);

    // sistema
    $usr = new UserController(); $crud($r, '/api/users', $usr);
    $aud = new AuditController(); $crud($r, '/api/audit-log', $aud);

    // ---- dispatch ----
    try {
        $res = $r->dispatch($method, $uri);
    } catch (\Throwable $e) {
        http_response_code(500);
        echo json_encode([
            'error' => 'server_error',
            'message' => $e->getMessage(),
        ]);
        return;
    }

    if ($res === null) {
        http_response_code(404);
        echo json_encode(['error' => 'route_not_found', 'path' => $uri]);
        return;
    }

    [$status, $body] = $res;
    http_response_code($status);
    if ($body === null) return;
    if (is_string($body)) { echo $body; return; }
    echo json_encode($body, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
}
