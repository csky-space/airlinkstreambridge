#pragma once

#ifndef TIMER_TPP
#define TIMER_TPP

#include "Timer.hpp"

void Timer::start() {
    start_time = clock::now();
    is_running = true;
}

void Timer::stop() {
    is_running = false;
}

template<typename Duration>
typename Duration::rep Timer::elapsed() const {
    if (!is_running) return 0;
    return std::chrono::duration_cast<Duration>(clock::now() - start_time).count();
}

void Timer::restart() {
    start();
}

#endif