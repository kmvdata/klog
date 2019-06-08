# klog
Golang logger package.

# Description

User to output logs in your application. 

# Features

- Print logs into stdout whether or not.
- Custom log file path.
- Custom max log file size.
- Compress log file into Date_Time.log.gzip when file size overload.
- Provide Info and Error level logs by default.

# Example

- Your func main() code:
		
		func main() {
			// Init
			klog.InitKLog("./log/my-log", 0, true)
		
			// Set Max Size
			// Or klog.SetMaxFileSizeMB(200) ==> Set to 200MB
  			klog.SetMaxFileSizeKB(200)
  		
  			// Test Code
			for i := 0; i < 10000; i++ {
				klog.Info.Printf("Hello %s", "World!")
				klog.Error.Printf("Hello %s", "Error!")
			}
		
			// Sleep make sure that uncompress *.log can be deleted.
			time.Sleep(time.Duration(2) * time.Second)
		}

- Result In StdOut Or Logfile:

		[Error] 2019/06/08 22:53:47.283476 Logger.go:206: Hello Error!
		[Info]  2019/06/08 22:53:47.283491 Logger.go:206: Hello World!

- Result In Directory:

		 > ls -lh ./log
		total 584
		-rw-r--r--  1 kmvdata.com  staff  60K Jun  8 22:56 2019-06-08_22-56-03.log.gzip
		-rw-r-----  1 kmvdata.com  staff 230K Jun  8 22:56 my-log.log
