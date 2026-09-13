<?php
require __DIR__.'/helpers.php';

check(parse_link('[[a]]') === ['map'=>'a','node'=>null], 'plain map link');
check(parse_link('[[a#b c]]') === ['map'=>'a','node'=>'b c'], 'map#node link');
check(parse_link('x [[a-b#n]] y') === ['map'=>'a-b','node'=>'n'], 'link embedded in text');
check(parse_link('no link') === null, 'no link');
check(parse_link('[[task:abcd1234]]') === null, 'task link ignored');
check(parse_link('[[a]] and [[b]]') === ['map'=>'a','node'=>null], 'first of two links');

$dir = sys_get_temp_dir().'/hmm_test_'.uniqid();
mkdir($dir);
file_put_contents("$dir/one.hmm", "one\n\tgo [[two#beta]]\n");
file_put_contents("$dir/two.hmm", "two\n\talpha\n\tbeta\n");

$mm['map_dir'] = $dir;
$mm['arguments']['filename'] = "$dir/one.hmm";
load_file($mm);

foreach ($mm['nodes'] as $id=>$node)
	if ($node['title'] === 'go [[two#beta]]')
		$mm['active_node'] = $id;

follow_link();
check(basename($mm['filename']) === 'two.hmm', 'follow_link opens target map');
check($mm['nodes'][$mm['active_node']]['title'] === 'beta', 'follow_link lands on target node');
check(count($mm['stack']) === 1, 'follow_link pushes stack');

go_back();
check(basename($mm['filename']) === 'one.hmm', 'go_back restores previous map');
check($mm['nodes'][$mm['active_node']]['title'] === 'go [[two#beta]]', 'go_back restores active node');
check(count($mm['stack']) === 0, 'go_back empties stack');

go_back();
check(basename($mm['filename']) === 'one.hmm', 'go_back on empty stack is a no-op');

open_map("$dir/two.hmm", 'missing');
check($mm['active_node'] === $mm['root_id'], 'missing target lands on root');
check($mm['nodes'][$mm['root_id']]['title'] === 'two', 'missing target does not fatal');

exec('rm -rf '.escapeshellarg($dir));
