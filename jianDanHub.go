package main

import (
	"sync"
	"log"
)

var urlSearchedMap = make(map[string]struct{})
var imageToDownloadMap = make(map[string]struct{})
var imageDownloadedMap = make(map[string]struct{})

var muxForUrl sync.RWMutex
var muxForImg sync.RWMutex
var muxForImgDownloaded sync.RWMutex

func Run() {
	jianDanFetcherChan := make(chan *JiandanFetcher) // job channel
	doneChan := make(chan struct{}) // 所有worker停止干活的条件是， 当某一个fetcher, 返回的所有imgurl链接，都已经在imgDownloaded中


	var wg sync.WaitGroup // 用于wait pageUrlChanChan的同步, 每向pageUrlChanChan/imgUrlChanChan中加1个channel, 则 wg + 1, 同步等待相关程序 执行完毕


	// 从根目录开始
	rootPageUrl := "http://www.jiandan.net/ooxx"
	go func() {
		jianDanFetcherChan <- NewJiandanFetcher(rootPageUrl)
	}()

	for i := 0; i < 100; i ++ {
		// 工人池
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case jiandanFetcher := <-jianDanFetcherChan:
					gotNewImg := false
					jiandanFetcher.Fetch()
					go func() {
						for pageUrl := range jiandanFetcher.UrlForPageChan {
							muxForUrl.RLock()
							if _, ok := urlSearchedMap[pageUrl]; !ok {
								muxForUrl.RUnlock()

								muxForUrl.Lock()
								urlSearchedMap[pageUrl] = struct{}{} //待下载
								muxForUrl.Unlock()

								jianDanFetcherChan <- NewJiandanFetcher(pageUrl)
							} else {
								muxForUrl.RUnlock()
							}
						}
					}()

					for imgUrl := range jiandanFetcher.UrlForImgChan {
						muxForImg.Lock()
						if _, ok := imageToDownloadMap[imgUrl]; !ok {
							// muxForImg.RUnlock()

							imageToDownloadMap[imgUrl] = struct{}{} //待下载
							muxForImg.Unlock()

							DownloadImg(imgUrl)

							muxForImgDownloaded.Lock()
							imageDownloadedMap[imgUrl] = struct{}{} // 已下载
							muxForImgDownloaded.Unlock()
							gotNewImg = true
						} else {
							muxForImg.Unlock()
						}
					}

					if !gotNewImg {
						// 如果当前页的图片都已经下载过了，则停止程序
						//close(doneChan)
					}
				case <-doneChan:
					log.Println("图片下载完成")
					return
				}

			}
		}()
	}

	wg.Wait()
	close(jianDanFetcherChan)
}
