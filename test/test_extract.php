<?php
require __DIR__.'/helpers.php';

$dir = sys_get_temp_dir().'/hmm_test_'.uniqid();
mkdir($dir);
file_put_contents("$dir/proj.hmm", "proj\n\tbig\n\t> some body\n\t\tleaf1\n\t\tleaf2\n\tother\n");

$mm['map_dir'] = $dir;
$mm['arguments']['filename'] = "$dir/proj.hmm";
load_file($mm);
$mm['active_node'] = 3; // big

check(extract_subtree(3, 'proj-big') === true, 'extract_subtree returns true');
check(file_get_contents("$dir/proj-big.hmm") === "proj-big\n> some body\n\tleaf1\n\tleaf2\n", 'extracted file content');
check($mm['nodes'][3]['title'] === '[[proj-big]]', 'node becomes link');
check($mm['nodes'][3]['children'] === [], 'children dropped');
check($mm['nodes'][3]['is_leaf'] === true, 'node is leaf');
check(($mm['nodes'][3]['body'] ?? '') === '', 'body cleared');
check($mm['modified'] === true, 'modified flag set');

save_vh($mm);
check(trim(file_get_contents("$dir/proj.hmm")) === "proj\n\t[[proj-big]]\n\tother", 'parent map saved with link');

$mm['nodes'] = [];
$mm['arguments']['filename'] = "$dir/proj-big.hmm";
load_file($mm);
check($mm['nodes'][$mm['root_id']]['title'] === 'proj-big', 'reloaded extracted map root');
$titles = [];
foreach ($mm['nodes'] as $n) $titles[] = $n['title'];
check(in_array('leaf1', $titles) && in_array('leaf2', $titles), 'reloaded extracted map children');

$mm['nodes'] = [];
$mm['arguments']['filename'] = "$dir/proj.hmm";
load_file($mm);
check($mm['nodes'][$mm['root_id']]['title'] === 'proj', 'reloaded parent map root');
$other_id = null;
$titles = [];
foreach ($mm['nodes'] as $id => $n)
{
	$titles[] = $n['title'];
	if ($n['title'] === 'other') $other_id = $id;
}
check(in_array('[[proj-big]]', $titles) && in_array('other', $titles), 'reloaded parent map children');

check(extract_subtree($other_id, 'proj-big') === false, 'existing target refused');
check(file_get_contents("$dir/proj-big.hmm") === "proj-big\n> some body\n\tleaf1\n\tleaf2\n", 'extracted file unchanged');

exec('rm -rf '.escapeshellarg($dir));
