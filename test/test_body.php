<?php
require __DIR__.'/helpers.php';

$map = "auth\n\tTokens\n\t\tJWT rotation\n\t\t> Rotate every 24h. Old key stays valid 1h.\n\t\t> See [[security]].\n\t\trefresh\n\t[[infra#redis]]\n";
$tmp = tempnam(sys_get_temp_dir(), 'hmm');
file_put_contents($tmp, $map);

$mm['arguments']['filename'] = $tmp;
load_file($mm);

// ids are sequential: auth=2 Tokens=3 JWT=4 refresh=5 infra=6
check($mm['nodes'][4]['title'] === 'JWT rotation', 'body lines do not become nodes');
check($mm['nodes'][4]['body'] === "Rotate every 24h. Old key stays valid 1h.\nSee [[security]].", 'body joined');
check(($mm['nodes'][5]['body'] ?? '') === '' && $mm['nodes'][5]['parent'] === 3, 'refresh: no body, sibling of JWT');
check(map_to_list($mm, $mm['root_id']) === $map, 'round trip');
check(map_node_count($tmp) === 5, 'node count skips body lines');

$n = list_to_map(["a", "> body", "\tb"], 0, 2);
check($n[3]['parent'] === 2, 'child after body line');
unlink($tmp);
