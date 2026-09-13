<?php
define('HMM_TEST', true);
$argv = ['hmm'];
require __DIR__.'/../hmm';

function check($cond, $msg)
{
	echo $cond ? "ok $msg\n" : "FAIL $msg\n";
	if (!$cond)
		exit(1);
}
