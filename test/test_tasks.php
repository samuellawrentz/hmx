<?php
require __DIR__.'/helpers.php';

$mm['tasks'] =
[
	 'aaaaaaaa' => ['done'=>false, 'overdue'=>false]
	,'bbbbbbbb' => ['done'=>true,  'overdue'=>false]
	,'cccccccc' => ['done'=>false, 'overdue'=>true]
];

check(node_text($mm, ['title'=>'[[task:aaaaaaaa]]']) === '☐ [[task:aaaaaaaa]]', 'pending task marker');
check(node_text($mm, ['title'=>'[[task:bbbbbbbb]]']) === '☑ [[task:bbbbbbbb]]', 'done task marker');
check(node_text($mm, ['title'=>'[[task:dddddddd]]']) === '☐ [[task:dddddddd]]', 'unknown uuid renders pending');
check(str_ends_with(node_text($mm, ['title'=>'x', 'body'=>'y']), ' …'), 'body marker appended');

check(parse_link('[[task:aaaaaaaa]]') === null, 'task link not a map link');

$export = json_encode
([
	['uuid'=>'cccccccc-0000-0000-0000-000000000000', 'status'=>'pending',   'due'=>'20000101T000000Z']
	,['uuid'=>'bbbbbbbb-0000-0000-0000-000000000000', 'status'=>'completed', 'due'=>'20000101T000000Z']
]);
$tasks = tasks_from_export($export);
check($tasks['cccccccc']['overdue'] === true, 'past due, pending -> overdue');
check($tasks['bbbbbbbb']['done'] === true && $tasks['bbbbbbbb']['overdue'] === false, 'completed task is done, not overdue');
check(tasks_from_export('[]') === [], 'empty export');
