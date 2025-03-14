#pragma once

#ifndef TIMER_HPP
#define TIMER_HPP

#include <chrono>
#include <thread>

class Timer {
    using clock = std::chrono::steady_clock;
    clock::time_point start_time;
    bool is_running = false;

public:
    Timer() = default;

    void start() {
        start_time = clock::now();
        is_running = true;
    }
    
    void stop() {
        is_running = false;
    }
    
    template<typename Duration>
    typename Duration::rep elapsed() const {
        if (!is_running) return 0;
        return std::chrono::duration_cast<Duration>(clock::now() - start_time).count();
    }
    
    void restart() {
        start();
    }
};

//#include "Timer.tpp"

#endif