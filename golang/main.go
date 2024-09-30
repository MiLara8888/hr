package main

import (
	"fmt"
	"sync"
	"time"
)

// Приложение эмулирует получение и обработку неких тасков. Пытается и получать, и обрабатывать в многопоточном режиме.
// Приложение должно генерировать таски 10 сек. Каждые 3 секунды должно выводить в консоль результат всех обработанных к этому моменту тасков (отдельно успешные и отдельно с ошибками).

// ЗАДАНИЕ: сделать из плохого кода хороший и рабочий - as best as you can.
// Важно сохранить логику появления ошибочных тасков.
// Важно оставить асинхронные генерацию и обработку тасков.
// Сделать правильную мультипоточность обработки заданий.
// Обновленный код отправить через pull-request в github
// Как видите, никаких привязок к внешним сервисам нет - полный карт-бланш на модификацию кода.

// A Ttype represents a meaninglessness of our life
type Task struct {
	id         int
	createAt   string // время создания
	leadAt     string // время выполнения
	taskResult []byte
}

// генерация задач n секунд
func taskGenerator(n int) (chan Task, chan Task) {

	var (
		taskList      = make(chan Task)
		errorTaskList = make(chan Task)
		done          = make(chan bool)
		to            = time.After(time.Duration(n) * time.Second)
	)

	go func(ch chan Task, chError chan Task) {
		for {
			select {
			case <-to:
				done <- true
				close(taskList)
				close(errorTaskList)
				return
			default:
				//TODO умышленно не трогаю эту логику
				ft := time.Now().Format(time.RFC3339)
				if time.Now().Nanosecond()%2 > 0 { //условие появления ошибочных задач
					errorTaskList <- Task{createAt: ft, id: int(time.Now().Unix())}
				}
				taskList <- Task{createAt: ft, id: int(time.Now().Nanosecond())} // передаем задачи на выполнение
			}
		}
	}(taskList, errorTaskList)

	return taskList, errorTaskList
}

func worker(jobs chan Task, resultTask chan Task, wg *sync.WaitGroup) {
	defer wg.Done()

	for a := range jobs {
		//TODO умышленно не трогаю эту логику
		tt, _ := time.Parse(time.RFC3339, a.createAt)
		if tt.After(time.Now().Add(-20 * time.Second)) {
			a.taskResult = []byte("task has been successed")
		} else {
			a.taskResult = []byte("something went wrong")
		}
		a.leadAt = time.Now().Format(time.RFC3339Nano)
		time.Sleep(time.Millisecond * 150)
		resultTask <- a
	}
}

func main() {

	//генерируем задачи 10 секунд
	task, taskError := taskGenerator(10)

	resultTask := make(chan Task)
	wg := &sync.WaitGroup{}
	workerPoolSize := 5

	//буфер для ошибочных результатов, защищаем мьютексом
	muErrorBuf := sync.Mutex{}
	errorBuf := []int{}

	//буфер для результатов, защищаем мьютексом
	muBuf := sync.Mutex{}
	resultBuf := [][]byte{}

	ticker := time.NewTicker(3 * time.Second)
	doneTicker := make(chan bool)

	wg.Add(workerPoolSize)
	for i := 0; i < workerPoolSize; i++ {
		go worker(task, resultTask, wg)
	}

	go func() {
		for {
			select {
			case n, ok := <-resultTask:
				if ok {
					muBuf.Lock()
					resultBuf = append(resultBuf, n.taskResult)
					muBuf.Unlock()
				}
			case n, ok := <-taskError:
				if ok {
					muErrorBuf.Lock()
					errorBuf = append(errorBuf, n.id)
					muErrorBuf.Unlock()
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-doneTicker:
				return
			case <-ticker.C:
				if len(resultBuf) == 0 && len(errorBuf) == 0 {
					doneTicker <- true
					ticker.Stop()
				}
				fmt.Println(errorBuf)
				muErrorBuf.Lock()
				errorBuf = nil
				muErrorBuf.Unlock()
				fmt.Println(resultBuf)
				muBuf.Lock()
				resultBuf = nil
				muBuf.Unlock()

			}
		}
	}()

	wg.Wait()

}
