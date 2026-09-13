<?php
define('HMX_TEST', true);
$argv = ['hmx'];
require __DIR__.'/../ref/hmx.php';

function check($cond, $msg)
{
	echo $cond ? "ok $msg\n" : "FAIL $msg\n";
	if (!$cond)
		exit(1);
}
