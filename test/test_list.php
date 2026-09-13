<?php
require __DIR__.'/helpers.php';

$dir = sys_get_temp_dir().'/hmm_test_'.uniqid();
mkdir($dir);
$mm['map_dir'] = $dir;

file_put_contents("$dir/inbox.hmm", "inbox\n\tx\n");
file_put_contents("$dir/a.hmm", "a\n\tsee [[b]]\n\tand [[b#x]]\n");
file_put_contents("$dir/b.hmm", "b\n\tx\n");
file_put_contents("$dir/c.hmm", "c\n\t> body only\n\tnope [[bb]]\n");

touch("$dir/inbox.hmm", time()-400);
touch("$dir/a.hmm", time()-300);
touch("$dir/b.hmm", time()-200);
touch("$dir/c.hmm", time()-100);

$rows = list_rows('');
check(count($rows) === 4, 'four maps listed');
check(array_column($rows, 'name') === ['inbox','c','b','a'], 'inbox pinned first, rest mtime desc');

check(array_column(list_rows('b'), 'name') === ['inbox','b'], 'filter is substring match on name');

foreach ($rows as $r)
	if ($r['name'] === 'c')
		check($r['count'] === 2, 'body line skipped from node count');

rename_map('b', 'z');
check(file_exists("$dir/z.hmm"), 'z.hmm exists after rename');
check(!file_exists("$dir/b.hmm"), 'b.hmm gone after rename');
check(file_get_contents("$dir/a.hmm") === "a\n\tsee [[z]]\n\tand [[z#x]]\n", 'inbound links rewritten');
check(file_get_contents("$dir/c.hmm") === "c\n\t> body only\n\tnope [[bb]]\n", 'unrelated link untouched');

rename_map('z', 'a');
check(file_get_contents("$dir/a.hmm") === "a\n\tsee [[z]]\n\tand [[z#x]]\n", 'rename refuses existing target');
check(file_exists("$dir/z.hmm"), 'z.hmm still exists after refused rename');

exec('rm -rf '.escapeshellarg($dir));
