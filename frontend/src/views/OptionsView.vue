<script setup lang="ts">
import { ui, go, setTheme, setOpacity } from "../store";
import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";

const GITHUB_URL = "https://github.com/viniciusbuscacio/go-calc";

function onThemeToggle(e: Event) {
  const dark = (e.target as HTMLInputElement).checked;
  setTheme(dark ? "dark" : "light");
}

function onOpacity(e: Event) {
  setOpacity(Number((e.target as HTMLInputElement).value));
}

function openGitHub() {
  BrowserOpenURL(GITHUB_URL);
}
</script>

<template>
  <div class="view view--panel">
    <div class="subheader">
      <button
        class="icon-btn"
        title="Back"
        data-testid="back"
        @click="go('calc')"
      >
        <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2Z" />
        </svg>
      </button>
      <h1 class="subheader-title">Settings</h1>
    </div>

    <div class="panel-body">
      <p class="section-title">Appearance</p>
      <label class="row switch-row">
        <span class="row-text">
          <span class="row-label">Dark mode</span>
          <span class="row-desc">Use the dark theme across the app</span>
        </span>
        <input
          type="checkbox"
          class="switch"
          role="switch"
          :checked="ui.theme === 'dark'"
          data-testid="theme-switch"
          @change="onThemeToggle"
        />
      </label>

      <div class="row">
        <span class="row-text">
          <span class="row-label">Transparency</span>
          <span class="row-desc">Window opacity — {{ ui.opacity }}%</span>
        </span>
        <input
          type="range"
          class="slider"
          min="20"
          max="100"
          step="1"
          :value="ui.opacity"
          data-testid="opacity-slider"
          @input="onOpacity"
        />
      </div>

      <p class="section-title">Advanced</p>
      <button class="row row--nav" data-testid="nav-api" @click="go('api')">
        <span class="row-text">
          <span class="row-label">REST API Server</span>
          <span class="row-desc">Expose the app over HTTP for automation</span>
        </span>
        <svg class="chevron" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path d="M10 6 8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6-6-6Z" />
        </svg>
      </button>

      <p class="section-title">About</p>
      <div class="about">
        <p class="about-desc">
          <strong>go-Calc</strong> is a Windows 11-style calculator built with
          Go + Wails and TypeScript. It also serves as a small template for
          cross-platform desktop apps.
        </p>
        <button class="row row--nav" data-testid="open-github" @click="openGitHub">
          <span class="row-text">
            <span class="row-label">GitHub</span>
            <span class="row-desc">github.com/viniciusbuscacio/go-calc</span>
          </span>
          <svg class="chevron" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M19 19H5V5h7V3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7h-2v7ZM14 3v2h3.59l-9.83 9.83 1.41 1.41L19 6.41V10h2V3h-7Z" />
          </svg>
        </button>
        <p v-if="ui.version" class="about-version" data-testid="app-version">
          Version {{ ui.version }}
        </p>
      </div>
    </div>
  </div>
</template>
