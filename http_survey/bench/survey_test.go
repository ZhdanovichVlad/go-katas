package bench_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ZhdanovichVlad/go-katas/http_survey/internal/semaphore"
	"github.com/ZhdanovichVlad/go-katas/http_survey/internal/workerpool"
)

// surveyFn — общий контракт обеих реализаций.
type surveyFn func(ctx context.Context, urls []string, parallel int, isMock bool) []int

// Слайс, а не map — чтобы порядок прогона бенчмарков был стабильным
// (по map Go итерируется в случайном порядке).
var implementations = []struct {
	name string
	fn   surveyFn
}{
	{"semaphore", semaphore.Survey},
	{"workerpool", workerpool.Survey},
}

var mockStatusCodes = map[int]struct{}{
	200: {},
	201: {},
	202: {},
	204: {},
	400: {},
	401: {},
	403: {},
	404: {},
	500: {},
	503: {},
}

// makeURLs возвращает n урлов, среди которых ровно `unique` различных.
// Это позволяет управлять процентом попаданий в кеш.
func makeURLs(n, unique int) []string {
	if unique <= 0 {
		unique = 1
	}
	urls := make([]string, n)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://example.test/%d", i%unique)
	}
	return urls
}

func assertMockResult(t *testing.T, got []int, wantLen int) {
	t.Helper()

	if len(got) != wantLen {
		t.Fatalf("len(got) = %d, want %d", len(got), wantLen)
	}

	for i, code := range got {
		if _, ok := mockStatusCodes[code]; !ok {
			t.Errorf("got[%d] = %d, want one of mock status codes", i, code)
		}
	}
}

// TestSurveyCorrectness — табличный тест: длина результата и значения статусов
// для каждой реализации на одних и тех же входах.
func TestSurveyCorrectness(t *testing.T) {
	cases := []struct {
		name     string
		urls     []string
		parallel int
		want     int
	}{
		{"empty", nil, 4, 0},
		{"one", []string{"https://a"}, 1, 1},
		{"unique_5", makeURLs(5, 5), 2, 5},
		{"dup_10_in_3", makeURLs(10, 3), 4, 10},
		{"parallel_gt_urls", makeURLs(3, 3), 16, 3},
		{"big", makeURLs(100, 70), 8, 100},
	}

	for _, impl := range implementations {
		for _, c := range cases {
			t.Run(impl.name+"/"+c.name, func(t *testing.T) {
				got := impl.fn(context.Background(), c.urls, c.parallel, true)
				assertMockResult(t, got, c.want)
			})
		}
	}
}

// TestSurveyImplementationsContract проверяет общий контракт обеих реализаций.
// Mock-клиент возвращает случайные HTTP-коды, поэтому реализации не обязаны
// возвращать побитово одинаковые срезы на разных запусках.
func TestSurveyImplementationsContract(t *testing.T) {
	urls := makeURLs(50, 10)
	ctx := context.Background()

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			got := impl.fn(ctx, urls, 8, true)
			assertMockResult(t, got, len(urls))
		})
	}
}

// BenchmarkSurvey гоняет обе реализации по одной матрице конфигов:
//   - размер пачки урлов,
//   - доля уникальных (управление процентом кеш-хитов),
//   - уровень параллелизма.
//
// ВАЖНО: мок client спит 100ms на каждый Get — поэтому абсолютные числа
// в первую очередь отражают latency мока, а не «чистый» оверхед примитивов
// синхронизации. Сравнивать имеет смысл relative-разницу между semaphore
// и workerpool в одной и той же ячейке матрицы.
func BenchmarkSurvey(b *testing.B) {
	type cfg struct {
		urls     int
		unique   int
		parallel int
	}
	cfgs := []cfg{
		{urls: 100, unique: 100, parallel: 1},
		{urls: 100, unique: 100, parallel: 8},
		{urls: 100, unique: 100, parallel: 32},
		{urls: 100, unique: 100, parallel: 128},
		{urls: 1000, unique: 1000, parallel: 8},
		{urls: 1000, unique: 1000, parallel: 64},
		{urls: 1000, unique: 1000, parallel: 256},
		{urls: 1000, unique: 50, parallel: 64}, // высокая доля кеш-хитов
	}

	for _, impl := range implementations {
		for _, c := range cfgs {
			name := fmt.Sprintf("%s/urls=%d/uniq=%d/par=%d",
				impl.name, c.urls, c.unique, c.parallel)
			urls := makeURLs(c.urls, c.unique)
			b.Run(name, func(b *testing.B) {
				ctx := context.Background()
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = impl.fn(ctx, urls, c.parallel, true)
				}
			})
		}
	}
}
