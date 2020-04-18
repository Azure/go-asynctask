package asynctask

// WaitAll block current thread til all task finished.
// it return immediately after receive first error.
func WaitAll(tasks ...*TaskStatus) error {
	errorCh := make(chan error)
	errorChClosed := false
	defer func() {
		errorChClosed = true
		close(errorCh)
	}()

	runningTasks := 0
	for _, tsk := range tasks {

		go func(tsk *TaskStatus) {
			_, err := tsk.Wait()
			if !errorChClosed {
				errorCh <- err
			}
		}(tsk)
		runningTasks++
	}

	for {
		select {
		case err := <-errorCh:
			runningTasks--
			if err != nil && isErrorReallyError(err) {
				return err
			}
			if runningTasks == 0 {
				return nil
			}
		}
	}
}
