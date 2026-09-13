<?php
require __DIR__.'/helpers.php';

check(count(array_unique($keybindings)) === 28, 'distinct functions bound');

foreach ($keybindings as $key => $fn)
	check(is_callable($fn), "callable: $fn");

check($keybindings["\n"] === 'enter_key', 'enter binds enter_key');
check($keybindings["\177"] === 'go_back', 'backspace binds go_back');
check(!isset($keybindings['x']), 'removed key x not bound');
