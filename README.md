# AsyncTask

![Build](https://github.com/Azure/go-asynctask/workflows/Go/badge.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/Azure/go-asynctask)](https://goreportcard.com/report/github.com/Azure/go-asynctask)
[![GoDoc](https://godoc.org/github.com/Azure/go-asynctask?status.svg)](https://godoc.org/github.com/Azure/go-asynctask)
[![Codecov](https://img.shields.io/codecov/c/github/Azure/go-asynctask)](https://codecov.io/gh/Azure/go-asynctask)

Simple mimik of async/await for those come from C# world, so you don't need to dealing with waitGroup/channel in golang.

also the result is strongTyped with go generics, no type assertion is needed.

few chaining method provided:
- ContinueWith: send task1's output to task2 as input, return reference to task2.
- AfterBoth : send output of taskA, taskB to taskC as input, return reference to taskC.
- WaitAll: all of the task have to finish to end the wait (with an option to fail early if any task failed)
- WaitAny: any of the task finish would end the wait

```golang
    // start task
    task := asynctask.Start(ctx, countingTask)
    
    // do something else
    somethingelse()
    
    // get the result
    rawResult, err := task.Wait()
    // or
    task.Cancel()
```

# Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
