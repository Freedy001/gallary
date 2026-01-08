import {defineStore} from 'pinia'
import {ref} from "vue";
import type {SearchParams} from "@/types";
import type {Fether} from "@/composables/useImageList.ts";
import {imageApi} from "@/api/image.ts";


export const useSearchStore = defineStore('search', () => {
  // Search state
  const searchDescription = ref('')
  const searchFilters = ref<SearchParams>({
    keyword: '',
    start_date: '',
    end_date: '',
    location: '',
    tags: [],  // 改为数组类型
    latitude: undefined,
    longitude: undefined,
    radius: 10,
  })


  const subscriber = new Map<string, (desc: string, fetcher: Fether) => void>()

  function subsribe(id: string, onSearch: (desc: string, fetcher: Fether) => void) {
    subscriber.set(id, onSearch)
  }

  function callSubscribers(param: SearchParams, desc: string) {
    subscriber.forEach(subscriber => subscriber(
      desc,
      async (page, size) => (await imageApi.search({...param, page, page_size: size})).data
    ))
  }

  return {
    // State
    searchDescription: searchDescription,
    searchFilters: searchFilters,
    subsribe: subsribe,
    callSubscribers: callSubscribers
  }
})
