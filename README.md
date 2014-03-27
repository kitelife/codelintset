## 安装配置过程

- 修改配置文件conf/app_conf.json
		
- 编译源码

		cd codelintset && go build main.go
		
- 为邮件发送脚本添加可执行权限

		chmod +x send_mail.py
		
- 克隆相关代码库：

        cd /tmp/
        git clone /home/git/repositories/shuangluo/distribution.git
        git clone /home/git/repositories/root/test.git
        
如果静态分析程序与git远程服务器在同一台机器上，就可以直接通过文件系统路径来克隆，这样也不用添加信任关系。

### 安装配置依赖

- 安装phpcs

		cd ~ && wget http://pear.php.net/go-pear.phar
		php go-pear.phar
		# 将/usr/local/php/bin添加到PATH环境变量中
		pear install PHP_CodeSniffer

这时执行phpcs可能会没有任何反应，从php的错误日志文件中查看错误原因。`/usr/local/php/bin/phpcs`是一个php脚本，其中有如下4行代码：

	if (is_file(dirname(__FILE__).'/../CodeSniffer/CLI.php') === true) {
	    include_once dirname(__FILE__).'/../CodeSniffer/CLI.php';
	} else {
	    include_once 'PHP/CodeSniffer/CLI.php';
	}
	
代码中使用了相对路径来查找真正的脚本程序，但实际安装的路径也许并不相同，所以先找出`CLI.php`程序的位置：

	find /usr/local/php -name CLI.php
	
结果如：

	/usr/local/php/lib/php/PHP/CodeSniffer/CLI.php
	/usr/local/php/lib/php/PEAR/Frontend/CLI.php
	
第一个结果就是PHP_CodeSniffer真正脚本的位置，根据该路径修改上述phpcs的4行代码修改为：

	if (is_file(dirname(__FILE__).'/../lib/php/PHP/CodeSniffer/CLI.php') === true) {
	    include_once dirname(__FILE__).'/../lib/php/PHP/CodeSniffer/CLI.php';
	} else {
	    include_once 'PHP/CodeSniffer/CLI.php';
	}
	
- 安装gjslint

		git clone https://github.com/youngsterxyf/closure-linter.git
		cd closure-linter && python setup.py install
	
- 安装flake8

		easy_install flake8
		
### 关于邮件发送脚本

邮件发送脚本调用了TOF2统一消息接口，所以需要为机器申请一个appkey才能发送邮件。


### 性能测试

	./main -cpuprofile=cpu.prof
	go tool pprof --functions svg main cpu.prof > functions.svg
	go tool pprof --lines svg main cpu.prof > lines.svg
	
帮助文档：

	go tool pprof help
