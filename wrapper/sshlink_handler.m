#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>

@interface SSHLinkHandler : NSObject <NSApplicationDelegate>
- (void)handleGetURLEvent:(NSAppleEventDescriptor *)event
           withReplyEvent:(NSAppleEventDescriptor *)replyEvent;
@end

@implementation SSHLinkHandler

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    // Register for GetURL Apple Events
    NSAppleEventManager *appleEventManager = [NSAppleEventManager sharedAppleEventManager];
    [appleEventManager setEventHandler:self
                           andSelector:@selector(handleGetURLEvent:withReplyEvent:)
                         forEventClass:kInternetEventClass
                            andEventID:kAEGetURL];
}

- (void)handleGetURLEvent:(NSAppleEventDescriptor *)event
           withReplyEvent:(NSAppleEventDescriptor *)replyEvent {

    // Extract URL from Apple Event
    NSAppleEventDescriptor *directObjectDescriptor = [event paramDescriptorForKeyword:keyDirectObject];
    NSString *urlString = [directObjectDescriptor stringValue];

    // Get path to our Go executable
    NSBundle *mainBundle = [NSBundle mainBundle];
    NSString *goExecPath = [[mainBundle bundlePath]
                           stringByAppendingPathComponent:@"Contents/MacOS/SSHLink-real"];

    // Launch Go application with URL
    NSTask *task = [[NSTask alloc] init];
    task.launchPath = goExecPath;
    task.arguments = @[urlString];

    [task launch];

    // Don't terminate - just stay alive and ready for next URL
    // This fixes the "every second click" issue
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    // Don't terminate when windows close - we're a background URL handler
    return NO;
}

- (void)applicationWillTerminate:(NSNotification *)notification {
    // Clean shutdown
}

@end

int main(int argc, const char * argv[]) {
    @autoreleasepool {
        // If we have command line arguments, pass them directly to Go executable
        if (argc > 1) {
            NSString *goExecPath = [[NSBundle mainBundle].bundlePath
                                   stringByAppendingPathComponent:@"Contents/MacOS/SSHLink-real"];

            NSMutableArray *args = [NSMutableArray array];
            for (int i = 1; i < argc; i++) {
                [args addObject:[NSString stringWithUTF8String:argv[i]]];
            }

            NSTask *task = [[NSTask alloc] init];
            task.launchPath = goExecPath;
            task.arguments = args;

            [task launch];
            [task waitUntilExit];
            return task.terminationStatus;
        }

        // No arguments - this is a URL handler launch
        // Stay alive as a background service for handling URLs
        NSApplication *app = [NSApplication sharedApplication];
        SSHLinkHandler *handler = [[SSHLinkHandler alloc] init];
        app.delegate = handler;

        // Run the app - it will stay alive and handle multiple URLs
        [app run];
    }
    return 0;
}
