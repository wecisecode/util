package spliter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Split(t *testing.T) {
	s := fmt.Sprint("[" + strings.Join(MQLSplit(`;begin batch
	select "
	end
	" from "
	begin
	batch
	";
end;begin
batch
	select "end" from "begin batch";
end
;
begin	batch
	select "end", end_of_select from 'begin " batch';
end
;
	select 双引号\转义； "a;\";\\;"; 单引号不转义； where x='a;'';c\';
;
空行
;
	-- 注释 \"'
	// 注释 ";
	/* ; */

	`),
		"\n-------------------------------------\n") + "]")
	fmt.Println(s)
	assert.Equal(t, `[
-------------------------------------
begin batch
	select "
	end
	" from "
	begin
	batch
	";
end
-------------------------------------
begin
batch
	select "end" from "begin batch";
end

-------------------------------------

begin	batch
	select "end", end_of_select from 'begin " batch';
end

-------------------------------------

	select 双引号\转义
-------------------------------------
 "a;\";\\;"
-------------------------------------
 单引号不转义
-------------------------------------
 where x='a;'';c\'
-------------------------------------


-------------------------------------

空行

-------------------------------------

	-- 注释 \"'
	// 注释 ";
	/* ; */

	]`, s)
}

func Test_SplitClean(t *testing.T) {
	s := fmt.Sprint("[" + strings.Join(MQLSplitClean(`;begin batch
	select "
	end
	" from "
	begin
	batch
	";
end;begin
batch
	select "end" from "begin batch";
end
;
begin	batch
	select "end", end_of_select from 'begin " batch';
end
;
	select 双引号\转义； "a;\";\\;"; 单引号不转义； where x='a;'';c\';
;
空行
;
	-- 注释 \"'
	// 注释 ";
	/* ; */

	`),
		"\n-------------------------------------\n") + "]")
	fmt.Println(s)
	assert.Equal(t, `[
-------------------------------------
begin batch
	select "
	end
	" from "
	begin
	batch
	";
end
-------------------------------------
begin
batch
	select "end" from "begin batch";
end

-------------------------------------

begin	batch
	select "end", end_of_select from 'begin " batch';
end

-------------------------------------

	select 双引号\转义
-------------------------------------
 "a;\";\\;"
-------------------------------------
 单引号不转义
-------------------------------------
 where x='a;'';c\'
-------------------------------------


-------------------------------------

空行

-------------------------------------

			

	]`, s)
}
