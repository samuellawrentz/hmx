<?php
require __DIR__.'/helpers.php';

$nodes = list_to_map(["a", "\tb"], 0, 2);
check(count($nodes) === 2, 'two nodes');
check($nodes[3]['title'] === 'b' && $nodes[3]['parent'] === 2, 'b is child of a');
