<?php
require __DIR__.'/helpers.php';

putenv("EDITOR=sh -c 'printf \"\\nadded line\\n\" >> \"\$0\"'");

$mm['arguments']['filename'] = __DIR__.'/fixtures/backend.hmm';
load_file($mm);

// ids are sequential: backend=2 auth=3 JWT=4 refresh=5
$mm['active_node'] = 5; // refresh, no existing body
edit_body();
check($mm['nodes'][5]['title'] === 'refresh', 'refresh: title unchanged');
check($mm['nodes'][5]['body'] === 'added line', 'refresh: body appended');
check($mm['modified'] === true, 'modified flag set');

$mm['active_node'] = 4; // JWT rotation, existing body
edit_body();
check($mm['nodes'][4]['title'] === 'JWT rotation', 'JWT rotation: title unchanged');
check($mm['nodes'][4]['body'] === "Rotate every 24h.\n\nadded line", 'JWT rotation: body appended after existing');
