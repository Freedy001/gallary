<template>
  <Listbox
    as="div"
    :model-value="modelValue"
    :disabled="disabled"
    @update:model-value="val => emit('update:modelValue', val)"
    v-slot="{ open, disabled: isDisabled }"
  >
    <div :class="['relative', open ? 'z-50' : '']">
      <!-- Label (Optional) -->
      <ListboxLabel v-if="label" class="block text-sm font-medium text-gray-300 mb-2">
        {{ label }}
      </ListboxLabel>

      <!-- Button -->
      <ListboxButton
        :class="[
          'relative w-full cursor-pointer rounded-lg bg-white/5 border border-white/10 py-2.5 pl-4 pr-10 text-left text-sm text-white shadow-sm transition-colors focus:outline-none focus:ring-1 focus:ring-primary-500 focus:border-primary-500',
          isDisabled ? 'opacity-50 cursor-not-allowed' : 'hover:bg-white/10',
          open ? 'border-primary-500 ring-1 ring-primary-500' : '',
          buttonClass
        ]"
      >
        <span class="block truncate">
          {{ selectedLabel || placeholder || '请选择' }}
        </span>
        <span class="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
          <ChevronUpDownIcon class="h-5 w-5 text-gray-400" aria-hidden="true" />
        </span>
      </ListboxButton>

      <!-- Options -->
      <transition
        leave-active-class="transition ease-in duration-100"
        leave-from-class="opacity-100"
        leave-to-class="opacity-0"
      >
        <ListboxOptions
          class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-xl bg-[#1A1A1A] border border-white/10 py-1 text-base shadow-lg ring-1 ring-black/5 focus:outline-none sm:text-sm backdrop-blur-xl"
          :class=" buttonClass"
        >
          <ListboxOption
            v-for="option in options"
            :key="option.value"
            :value="option.value"
            :disabled="option.disabled"
            as="template"
            v-slot="{ active, selected, disabled: optionDisabled }"
          >
            <li
              :class="[
                active ? 'bg-primary-500/20 text-primary-300' : 'text-gray-300',
                optionDisabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
                'relative cursor-default select-none py-2.5 pl-10 pr-4 transition-colors'
              ]"
            >
              <Tooltip :content="option.label" class="w-full" show-only-if-truncated>
                <span
                  :class="[
                    selected ? 'font-medium text-primary-400' : 'font-normal',
                    'block truncate'
                  ]"
                >
                  {{ option.label }}
                </span>
              </Tooltip>
              <span
                v-if="selected"
                class="absolute inset-y-0 left-0 flex items-center pl-3 text-primary-500"
              >
                <CheckIcon class="h-5 w-5" aria-hidden="true" />
              </span>
            </li>
          </ListboxOption>

          <!-- Empty State -->
          <div v-if="options.length === 0" class="py-3 px-4 text-gray-500 text-sm text-center">
            暂无选项
          </div>
        </ListboxOptions>
      </transition>
    </div>
  </Listbox>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {Listbox, ListboxButton, ListboxLabel, ListboxOption, ListboxOptions,} from '@headlessui/vue'
import {CheckIcon, ChevronUpDownIcon} from '@heroicons/vue/20/solid'
import Tooltip from './Tooltip.vue'

export interface SelectOption {
  label: string
  value: string | number
  disabled?: boolean
  [key: string]: any
}

const props = defineProps<{
  modelValue: string | number | null | undefined
  options: SelectOption[]
  label?: string
  placeholder?: string
  disabled?: boolean
  buttonClass?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | number): void
}>()

const selectedLabel = computed(() => {
  const option = props.options.find(opt => opt.value === props.modelValue)
  return option ? option.label : ''
})
</script>
