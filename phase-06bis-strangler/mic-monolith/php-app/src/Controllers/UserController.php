<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class UserController extends BaseController
{
    protected function type(): string { return 'utente'; }

    protected function toDto(array $r): array
    {
        return [
            'id'      => $r['id'],
            'email'   => $r['code'],
            'nome'    => $r['name'],
            'ruolo'   => $r['text_2'],
            'status'  => $r['status'],
        ];
    }

    protected function fromPayload(array $b): array
    {
        return [
            'code'    => $b['email'] ?? $b['code'] ?? null,
            'name'    => $b['nome']  ?? $b['name'] ?? null,
            'text_1'  => $b['email'] ?? null,
            'text_2'  => $b['ruolo'] ?? 'operatore',
            'status'  => $b['status'] ?? 'attivo',
            'payload' => ['hash' => '$2y$10$placeholder'],
        ];
    }
}
