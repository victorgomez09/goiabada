<script setup lang="ts">
import { useVuelidate } from "@vuelidate/core";
import { email, helpers, required } from "@vuelidate/validators";
import { reactive, toRaw } from "vue";

import { Login } from "../models/auth.model";
import { post } from "../utils/fetch.util";

const formValues = reactive<Login>({
  email: "",
  password: "",
});

const validators = {
  email: {
    required: helpers.withMessage("Email is required", required),
    email: helpers.withMessage("Email is not valid", email),
  },
  password: {
    required: helpers.withMessage("Password is required", required),
  },
};

const $v = useVuelidate(validators, formValues);

const handleSubmit = () => {
  //prevent default
  console.log("values", toRaw(formValues));
  const fetch = post<any, any>("/auth/pwd", formValues);

  try {
    const response = fetch;
    console.log("response", response);
  } catch (error) {
    console.log("error", error);
  }
};
</script>

<template>
  <div class="flex flex-1 justify-center items-center w-full h-full">
    <div class="card shadow-md bg-base-100 w-4/12">
      <div class="card-body">
        <h2 class="card-title">Login</h2>
        <form class="flex flex-col gap-4" @submit.prevent="handleSubmit">
          <div class="form-control">
            <label class="label">
              <span class="label-text">Email</span>
            </label>
            <input
              @blur="$v.email.$touch()"
              v-model="formValues.email"
              type="text"
              placeholder="email"
              class="input input-bordered"
              :class="{ 'input-error': $v.email.$error }"
            />
            <label
              v-if="$v.email.$error"
              class="label"
              :class="{ 'label-error': $v.email.$error }"
            >
              <span
                class="label-text-alt text-error"
                v-for="error in $v.email.$errors"
              >
                {{ error.$message }}
              </span>
            </label>
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">Password</span>
            </label>
            <input
              @blur="$v.password.$touch()"
              v-model="formValues.password"
              type="password"
              placeholder="password"
              class="input input-bordered"
              :class="{ 'input-error': $v.password.$error }"
            />
            <label
              v-if="$v.password.$error"
              class="label"
              :class="{ 'label-error': $v.password.$error }"
            >
              <span
                class="label-text-alt text-error"
                v-for="error in $v.password.$errors"
              >
                {{ error.$message }}
              </span>
            </label>
          </div>
          <div class="form-control mt-6">
            <button
              type="submit"
              class="btn btn-primary"
              :disabled="$v.$invalid"
            >
              Login
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
